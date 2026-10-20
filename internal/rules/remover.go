package rules

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	archfs "github.com/EnzoCaetano015/Archbase/internal/filesystem"
)

var ruleMarkerPattern = regexp.MustCompile(`^<!-- archbase:rule ([a-z0-9][a-z0-9._-]*/[a-z0-9][a-z0-9._-]*@[a-z0-9][a-z0-9._-]*) (export|start|end) -->$`)

type RemoveOptions struct {
	Destination string
	All         bool
}

type RemoveResult struct {
	Paths         []string
	RuleIDs       []string
	LegacyExports int
}

type removalAction struct {
	path     string
	original []byte
	updated  []byte
	remove   bool
	ruleIDs  []string
	legacy   bool
}

// Remove deletes owned Cursor and Copilot exports and removes only managed
// Archbase blocks from AGENTS.md files.
func (e *Exporter) Remove(rawID string, options RemoveOptions) (RemoveResult, error) {
	if options.Destination == "" {
		options.Destination = "."
	}
	if options.All {
		if rawID != "" {
			return RemoveResult{}, errors.New("a rule ID cannot be combined with --all")
		}
	} else if _, err := ParseRuleID(rawID); err != nil {
		return RemoveResult{}, err
	}

	root, err := filepath.Abs(options.Destination)
	if err != nil {
		return RemoveResult{}, &archfs.PathError{Op: "resolve removal destination", Path: options.Destination, Err: err}
	}
	exists, err := e.fs.Exists(root)
	if err != nil {
		return RemoveResult{}, err
	}
	if !exists {
		return RemoveResult{}, &archfs.PathError{Op: "remove rules", Path: root, Err: fs.ErrNotExist}
	}
	if err := e.validateRoot(root); err != nil {
		return RemoveResult{}, err
	}
	for _, relative := range []string{filepath.Join(".cursor", "rules"), filepath.Join(".github", "instructions")} {
		target := filepath.Join(root, relative)
		if err := e.rejectSymlinkPath(root, target); err != nil {
			return RemoveResult{}, err
		}
	}

	actions := make([]removalAction, 0)
	err = e.fs.WalkDir(root, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return &archfs.PathError{Op: "scan rule exports", Path: current, Err: walkErr}
		}
		if current == root {
			return nil
		}
		relative, err := filepath.Rel(root, current)
		if err != nil {
			return &archfs.PathError{Op: "resolve rule export", Path: current, Err: err}
		}
		slashed := filepath.ToSlash(relative)
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == ".archbase" || entry.Name() == "node_modules" {
				return fs.SkipDir
			}
			return nil
		}
		candidate := isAgentsPath(slashed) || isOwnedExportPath(slashed)
		if !candidate {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return &archfs.PathError{Op: "remove rule export", Path: current, Err: archfs.ErrSymlink}
		}
		info, err := entry.Info()
		if err != nil {
			return &archfs.PathError{Op: "inspect rule export", Path: current, Err: err}
		}
		if !info.Mode().IsRegular() {
			return &archfs.PathError{Op: "remove rule export", Path: current, Err: errors.New("target is not a regular file")}
		}
		content, err := e.fs.ReadFile(current)
		if err != nil {
			return err
		}
		if isAgentsPath(slashed) {
			updated, ids, changed, err := removeManagedBlocks(content, rawID, options.All)
			if err != nil {
				return &archfs.PathError{Op: "remove managed rule block", Path: current, Err: err}
			}
			if changed {
				actions = append(actions, removalAction{path: current, original: content, updated: updated, ruleIDs: ids})
			}
			return nil
		}
		id, owned, err := ownedExportRuleID(content)
		if err != nil {
			return &archfs.PathError{Op: "inspect managed rule export", Path: current, Err: err}
		}
		if owned && (options.All || id == rawID) {
			actions = append(actions, removalAction{path: current, original: content, remove: true, ruleIDs: []string{id}})
		} else if !owned && isLegacyManagedExport(content, slashed) && (options.All || slashed == expectedExportPath(rawID, slashed)) {
			ids := []string(nil)
			if !options.All {
				ids = []string{rawID}
			}
			actions = append(actions, removalAction{path: current, original: content, remove: true, ruleIDs: ids, legacy: true})
		}
		return nil
	})
	if err != nil {
		return RemoveResult{}, err
	}

	sort.Slice(actions, func(left, right int) bool { return actions[left].path < actions[right].path })
	applied := make([]removalAction, 0, len(actions))
	for _, action := range actions {
		var actionErr error
		if action.remove {
			actionErr = e.fs.Remove(action.path)
		} else {
			actionErr = e.fs.WriteFileAtomic(action.path, action.updated, true)
		}
		if actionErr != nil {
			rollbackErr := e.rollbackRemovals(append(applied, action))
			if rollbackErr != nil {
				return RemoveResult{}, fmt.Errorf("remove rules at %q: %w (rollback failed: %v)", action.path, actionErr, rollbackErr)
			}
			return RemoveResult{}, fmt.Errorf("remove rules at %q: %w", action.path, actionErr)
		}
		applied = append(applied, action)
	}

	result := RemoveResult{Paths: make([]string, 0, len(actions))}
	ids := map[string]bool{}
	for _, action := range actions {
		result.Paths = append(result.Paths, action.path)
		for _, id := range action.ruleIDs {
			ids[id] = true
		}
		if action.legacy {
			result.LegacyExports++
		}
	}
	for id := range ids {
		result.RuleIDs = append(result.RuleIDs, id)
	}
	sort.Strings(result.RuleIDs)
	return result, nil
}

func (e *Exporter) rollbackRemovals(actions []removalAction) error {
	var failures []error
	for index := len(actions) - 1; index >= 0; index-- {
		action := actions[index]
		exists, err := e.fs.Exists(action.path)
		if err == nil {
			err = e.fs.EnsureDir(filepath.Dir(action.path))
		}
		if err == nil {
			err = e.fs.WriteFileAtomic(action.path, action.original, exists)
		}
		if err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func isAgentsPath(relative string) bool {
	return filepath.Base(filepath.FromSlash(relative)) == "AGENTS.md"
}

func isOwnedExportPath(relative string) bool {
	if strings.HasPrefix(relative, ".cursor/rules/") && strings.HasSuffix(relative, ".mdc") {
		return !strings.Contains(strings.TrimPrefix(relative, ".cursor/rules/"), "/")
	}
	if strings.HasPrefix(relative, ".github/instructions/") && strings.HasSuffix(relative, ".instructions.md") {
		return !strings.Contains(strings.TrimPrefix(relative, ".github/instructions/"), "/")
	}
	return false
}

func expectedExportPath(id, candidate string) string {
	if strings.HasPrefix(candidate, ".cursor/rules/") {
		return filepath.ToSlash(filepath.Join(".cursor", "rules", ruleSlug(id)+".mdc"))
	}
	return filepath.ToSlash(filepath.Join(".github", "instructions", ruleSlug(id)+".instructions.md"))
}

func isLegacyManagedExport(content []byte, relative string) bool {
	text := strings.ReplaceAll(string(content), "\r\n", "\n")
	if strings.Contains(text, "<!-- archbase:rule ") {
		return false
	}
	const generatedFooter = "## Pattern resolution\n\nUse `arc resolve <target-path>` to find the nearest active local pattern. Use `arc inspect <pattern-id>` to inspect a referenced pattern before changing code.\n\n"
	if !strings.HasSuffix(text, generatedFooter) || !strings.Contains(text, "\n## Applicable scopes\n\n") || !strings.Contains(text, "\n## Architecture rules\n\n") {
		return false
	}
	if strings.HasPrefix(relative, ".cursor/rules/") {
		return strings.HasPrefix(text, "---\ndescription: ") && strings.Contains(text, "\nglobs:\n") && strings.Contains(text, "\nalwaysApply: false\n---\n\n")
	}
	return strings.HasPrefix(text, "---\napplyTo: ") && strings.Contains(text, "\n---\n\n")
}

func ownedExportRuleID(content []byte) (string, bool, error) {
	var found string
	for _, line := range splitLines(content) {
		trimmed := trimLineEnding(line)
		if !strings.Contains(trimmed, "<!-- archbase:rule ") {
			continue
		}
		id, kind, err := parseRuleMarker(trimmed)
		if err != nil || kind != "export" {
			if err == nil {
				err = fmt.Errorf("unexpected %s marker in owned export", kind)
			}
			return "", false, err
		}
		if found != "" {
			return "", false, errors.New("managed export marker is duplicated")
		}
		found = id
	}
	return found, found != "", nil
}

func removeManagedBlocks(content []byte, requestedID string, all bool) ([]byte, []string, bool, error) {
	var output strings.Builder
	activeID := ""
	removing := false
	counts := map[string]int{}
	removed := map[string]bool{}
	for _, line := range splitLines(content) {
		trimmed := trimLineEnding(line)
		if !strings.Contains(trimmed, "<!-- archbase:rule ") {
			if !removing {
				output.WriteString(line)
			}
			continue
		}
		id, kind, err := parseRuleMarker(trimmed)
		if err != nil {
			return nil, nil, false, err
		}
		switch kind {
		case "start":
			if activeID != "" {
				return nil, nil, false, errors.New("managed rule blocks overlap")
			}
			counts[id]++
			if counts[id] > 1 {
				return nil, nil, false, fmt.Errorf("managed block for %s is duplicated", id)
			}
			activeID = id
			removing = all || id == requestedID
			if !removing {
				output.WriteString(line)
			}
		case "end":
			if activeID == "" || activeID != id {
				return nil, nil, false, fmt.Errorf("managed block end for %s is unmatched", id)
			}
			if removing {
				removed[id] = true
			} else {
				output.WriteString(line)
			}
			activeID = ""
			removing = false
		case "export":
			return nil, nil, false, errors.New("export ownership marker is invalid in AGENTS.md")
		}
	}
	if activeID != "" {
		return nil, nil, false, fmt.Errorf("managed block for %s is incomplete", activeID)
	}
	ids := make([]string, 0, len(removed))
	for id := range removed {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return []byte(output.String()), ids, len(ids) > 0, nil
}

func parseRuleMarker(line string) (string, string, error) {
	matches := ruleMarkerPattern.FindStringSubmatch(line)
	if matches == nil {
		return "", "", fmt.Errorf("invalid managed rule marker %q", line)
	}
	if _, err := ParseRuleID(matches[1]); err != nil {
		return "", "", err
	}
	return matches[1], matches[2], nil
}

func splitLines(content []byte) []string {
	text := string(content)
	if text == "" {
		return nil
	}
	lines := strings.SplitAfter(text, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func trimLineEnding(line string) string {
	return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
}
