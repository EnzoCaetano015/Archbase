package rules

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
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

func (e *Exporter) Remove(id string, options RemoveOptions) (RemoveResult, error) {
	return e.RemoveContext(context.Background(), id, options)
}

func (e *Exporter) RemoveContext(ctx context.Context, id string, options RemoveOptions) (RemoveResult, error) {
	if options.All {
		if id != "" {
			return RemoveResult{}, errors.New("a rule ID cannot be combined with --all")
		}
	} else if _, err := ParseRuleID(id); err != nil {
		return RemoveResult{}, err
	}
	if options.Destination == "" {
		options.Destination = "."
	}
	root, err := filepath.Abs(options.Destination)
	if err != nil {
		return RemoveResult{}, err
	}
	exists, err := e.fs.Exists(root)
	if err != nil {
		return RemoveResult{}, err
	}
	if !exists {
		return RemoveResult{}, fmt.Errorf("remove rules at %q: %w", root, fs.ErrNotExist)
	}
	if err = e.validateRoot(root); err != nil {
		return RemoveResult{}, err
	}
	unlock, err := archfs.AcquireWorkspaceLock(ctx, root, e.fs)
	if err != nil {
		return RemoveResult{}, err
	}
	defer unlock()
	for _, p := range []string{".cursor/rules", ".github/instructions"} {
		if err = e.rejectSymlinkPath(root, filepath.Join(root, filepath.FromSlash(p))); err != nil {
			return RemoveResult{}, err
		}
	}
	documents, files, err := e.loadDocuments(ctx, root)
	if err != nil {
		return RemoveResult{}, err
	}
	changes, err := e.scanLegacy(ctx, root, id, options.All)
	if err != nil {
		return RemoveResult{}, err
	}
	for found, c := range files {
		if options.All || found == id {
			c.remove = true
			c.ids = []string{found}
			changes[c.path] = c
			delete(documents, found)
		}
	}
	if err = e.synchronizeIndices(root, "", documents, changes); err != nil {
		return RemoveResult{}, err
	}
	paths, err := e.applyChanges(ctx, root, changes)
	if err != nil {
		return RemoveResult{}, err
	}
	result := RemoveResult{Paths: paths}
	ids := map[string]bool{}
	for _, p := range paths {
		c := changes[p]
		for _, found := range c.ids {
			ids[found] = true
		}
		if c.legacy {
			result.LegacyExports++
		}
	}
	if len(paths) > 0 && !options.All {
		ids[id] = true
	}
	for found := range ids {
		result.RuleIDs = append(result.RuleIDs, found)
	}
	sort.Strings(result.RuleIDs)
	return result, nil
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
