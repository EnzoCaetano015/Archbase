package skillbundle

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os/exec"
	"path"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing/fstest"

	"github.com/EnzoCaetano015/Archbase/internal/contracts"
	patternbundle "github.com/EnzoCaetano015/Archbase/internal/patterns"
	"github.com/goccy/go-yaml"
)

type indexEntry struct {
	ID      string `yaml:"id"`
	Version string `yaml:"version"`
	Path    string `yaml:"path"`
}
type catalogIndex struct {
	SchemaVersion int          `yaml:"schemaVersion"`
	Patterns      []indexEntry `yaml:"patterns"`
	Rules         []indexEntry `yaml:"rules"`
}

func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

func gitSnapshot(ctx context.Context, root, commit string) (fstest.MapFS, error) {
	if !commitID.MatchString(commit) {
		return nil, fmt.Errorf("catalog revision must be a full commit ID")
	}
	// Read Git blobs, not checkout bytes: CRLF conversion and dirty files cannot alter inputs.
	tree, err := exec.CommandContext(ctx, "git", "-C", root, "ls-tree", "-r", "-z", commit, "--", "LICENSE", "registry").Output()
	if err != nil {
		return nil, fmt.Errorf("read catalog tree: %w", err)
	}
	type blob struct{ name, id string }
	var blobs []blob
	var requests strings.Builder
	for _, entry := range bytes.Split(tree, []byte{0}) {
		if len(entry) == 0 {
			continue
		}
		parts := bytes.SplitN(entry, []byte{'\t'}, 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("malformed catalog Git entry")
		}
		fields := strings.Fields(string(parts[0]))
		name := string(parts[1])
		if len(fields) != 3 || fields[1] != "blob" || fields[0] != "100644" && fields[0] != "100755" {
			return nil, fmt.Errorf("catalog symlink or non-regular blob: %s", name)
		}
		if err := safePath(name); err != nil {
			return nil, err
		}
		blobs = append(blobs, blob{name, fields[2]})
		requests.WriteString(fields[2] + "\n")
	}
	command := exec.CommandContext(ctx, "git", "-C", root, "cat-file", "--batch")
	command.Stdin = strings.NewReader(requests.String())
	raw, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("read catalog blobs: %w", err)
	}
	reader := bufio.NewReader(bytes.NewReader(raw))
	files := fstest.MapFS{}
	for _, blob := range blobs {
		header, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		fields := strings.Fields(header)
		if len(fields) != 3 || fields[0] != blob.id || fields[1] != "blob" {
			return nil, fmt.Errorf("invalid catalog blob %s", blob.name)
		}
		size, err := strconv.Atoi(fields[2])
		if err != nil || size < 0 || size > MaxFileBytes {
			return nil, fmt.Errorf("invalid catalog blob size: %s", blob.name)
		}
		data := make([]byte, size)
		if _, err = io.ReadFull(reader, data); err != nil {
			return nil, err
		}
		separator, err := reader.ReadByte()
		if err != nil || separator != '\n' {
			return nil, fmt.Errorf("invalid catalog blob framing")
		}
		files[blob.name] = &fstest.MapFile{Data: data, Mode: 0644}
	}
	return files, nil
}

// CreateLock records all canonical rules and their transitive pattern sources.
// The only network operation is an explicit checkout/fetch performed by the caller.
func CreateLock(ctx context.Context, root, tag, commit string) (CatalogLock, error) {
	lock, _, err := createLock(ctx, root, tag, commit)
	return lock, err
}

func createLock(ctx context.Context, root, tag, commit string) (CatalogLock, fstest.MapFS, error) {
	lock := CatalogLock{}
	if !commitID.MatchString(commit) || !strings.HasPrefix(tag, "v") || !semver.MatchString(strings.TrimPrefix(tag, "v")) {
		return lock, nil, fmt.Errorf("invalid tag or commit")
	}
	resolved, err := exec.CommandContext(ctx, "git", "-C", root, "rev-parse", "--verify", "refs/tags/"+tag+"^{commit}").Output()
	if err != nil || strings.TrimSpace(string(resolved)) != commit {
		return lock, nil, fmt.Errorf("catalog tag %s does not identify %s", tag, commit)
	}
	snapshot, err := gitSnapshot(ctx, root, commit)
	if err != nil {
		return lock, nil, err
	}
	lock, err = lockSnapshot(snapshot, tag, commit)
	return lock, snapshot, err
}

func lockSnapshot(snapshot fstest.MapFS, tag, commit string) (CatalogLock, error) {
	lock := CatalogLock{SchemaVersion: 1, Repository: CatalogRepository, Tag: tag, Commit: commit}
	var err error
	files := fstest.MapFS{}
	read := func(file string) ([]byte, error) {
		if data, ok := files[file]; ok {
			return data.Data, nil
		}
		data, err := fs.ReadFile(snapshot, file)
		if err == nil {
			files[file] = &fstest.MapFile{Data: data, Mode: 0644}
		}
		return data, err
	}
	for _, file := range []string{"LICENSE", "registry/index.yaml", "registry/rules/index.yaml"} {
		if _, err = read(file); err != nil {
			return lock, err
		}
	}
	var patterns, rules catalogIndex
	if err = yaml.Unmarshal(files["registry/index.yaml"].Data, &patterns); err != nil {
		return lock, err
	}
	if err = yaml.Unmarshal(files["registry/rules/index.yaml"].Data, &rules); err != nil {
		return lock, err
	}
	if patterns.SchemaVersion != 1 || rules.SchemaVersion != 1 || len(rules.Rules) == 0 {
		return lock, fmt.Errorf("invalid catalog indices")
	}
	patternByID := map[string]indexEntry{}
	previous := ""
	for _, entry := range patterns.Patterns {
		if entry.ID <= previous {
			return lock, fmt.Errorf("pattern index must be unique and sorted: %s", entry.ID)
		}
		if err = safePath(entry.Path); err != nil {
			return lock, err
		}
		patternByID[entry.ID] = entry
		previous = entry.ID
	}
	previous = ""
	for _, entry := range rules.Rules {
		if entry.ID <= previous {
			return lock, fmt.Errorf("rule index must be unique and sorted: %s", entry.ID)
		}
		previous = entry.ID
		if err = safePath(entry.Path); err != nil {
			return lock, err
		}
		rulePath := path.Join("registry/rules", entry.Path, "rule.yaml")
		if _, err = read(rulePath); err != nil {
			return lock, err
		}
		definition, err := contracts.LoadRuleFS(files, rulePath)
		if err != nil {
			return lock, err
		}
		if definition.ID != entry.ID || definition.Version != entry.Version {
			return lock, fmt.Errorf("rule index metadata mismatch: %s", entry.ID)
		}
		locked := CatalogEntry{ID: entry.ID, Version: entry.Version, Path: rulePath, Patterns: []string{}}
		seen := map[string]bool{}
		for _, scope := range definition.Scopes {
			if seen[scope.Pattern] {
				continue
			}
			seen[scope.Pattern] = true
			pattern, ok := patternByID[scope.Pattern]
			if !ok {
				return lock, fmt.Errorf("rule %s: missing pattern %s", entry.ID, scope.Pattern)
			}
			manifestPath := path.Join("registry", pattern.Path, "manifest.yaml")
			if _, err = read(manifestPath); err != nil {
				return lock, err
			}
			manifest, err := contracts.LoadManifestFS(files, manifestPath)
			if err != nil {
				return lock, err
			}
			if manifest.ID != pattern.ID || manifest.Version != pattern.Version {
				return lock, fmt.Errorf("pattern index metadata mismatch: %s", pattern.ID)
			}
			bundle, err := patternbundle.Load(snapshot, path.Join("registry", pattern.Path))
			if err != nil {
				return lock, err
			}
			for _, item := range bundle.Files {
				if !item.Present {
					continue
				}
				if err = safePath(item.Spec.Source); err != nil {
					return lock, err
				}
				data, err := read(path.Join("registry", pattern.Path, item.Spec.Source))
				if err != nil || len(data) == 0 {
					return lock, fmt.Errorf("pattern %s: missing or empty declared file %s: %v", pattern.ID, item.Spec.Source, err)
				}
			}
			locked.Patterns = append(locked.Patterns, scope.Pattern)
		}
		sort.Strings(locked.Patterns)
		lock.Rules = append(lock.Rules, locked)
	}
	for file, data := range files {
		lock.Files = append(lock.Files, LockedFile{Path: file, SHA256: digest(data.Data)})
	}
	sort.Slice(lock.Files, func(i, j int) bool { return lock.Files[i].Path < lock.Files[j].Path })
	return lock, nil
}

func verifyCatalog(ctx context.Context, root string, lock CatalogLock) (fs.FS, error) {
	actual, snapshot, err := createLock(ctx, root, lock.Tag, lock.Commit)
	if err != nil {
		return nil, err
	}
	// Compare the complete graph, not just individually declared hashes: dropping
	// a required file or inventing an ID in a lock must not weaken validation.
	if !reflect.DeepEqual(actual, lock) {
		return nil, fmt.Errorf("catalog lock differs from canonical indices, IDs, versions or SHA-256 hashes")
	}
	files := fstest.MapFS{}
	for _, item := range lock.Files {
		data, err := fs.ReadFile(snapshot, item.Path)
		if err != nil {
			return nil, err
		}
		if digest(data) != item.SHA256 {
			return nil, fmt.Errorf("catalog %s: SHA-256 mismatch", item.Path)
		}
		files[item.Path] = &fstest.MapFile{Data: data, Mode: 0644}
	}
	return files, nil
}

func generatedCatalog(ctx context.Context, root string, lock CatalogLock, selected []string) (map[string][]byte, error) {
	snapshot, err := verifyCatalog(ctx, root, lock)
	if err != nil {
		return nil, err
	}
	files := map[string][]byte{}
	license, err := fs.ReadFile(snapshot, "LICENSE")
	if err != nil {
		return nil, err
	}
	files["LICENSE.archbase-MIT.txt"] = license
	index := fmt.Sprintf("# Archbase catalog\n\nGenerated from %s, tag %s, commit `%s`.\n\nCanonical references remain MIT licensed; see [license](../../LICENSE.archbase-MIT.txt).\n\n", lock.Repository, lock.Tag, lock.Commit)
	entries := map[string]CatalogEntry{}
	for _, entry := range lock.Rules {
		entries[entry.ID] = entry
	}
	previous := ""
	for _, id := range selected {
		if id <= previous {
			return nil, fmt.Errorf("manifest rules must be unique and sorted: %s", id)
		}
		previous = id
		entry, ok := entries[id]
		if !ok {
			return nil, fmt.Errorf("catalog: rule %s does not exist at %s", id, lock.Commit)
		}
		rule, err := contracts.LoadRuleFS(snapshot, entry.Path)
		if err != nil {
			return nil, err
		}
		name := strings.ReplaceAll(strings.ReplaceAll(id, "/", "-"), "@", "-") + ".md"
		index += fmt.Sprintf("- [%s](%s): %s\n", id, name, rule.Description)
		doc := fmt.Sprintf("# %s\n\nRuleID: `%s`  \nVersion: `%s`  \nSource: %s/blob/%s/%s  \nLicense: MIT\n\n%s\n\n## Restrictions\n\n", rule.Name, id, rule.Version, lock.Repository, lock.Commit, entry.Path, rule.Description)
		for _, restriction := range rule.Rules {
			doc += "- " + restriction + "\n"
		}
		doc += "\n## Pattern mappings\n\n"
		for _, scope := range rule.Scopes {
			doc += fmt.Sprintf("- `%s` → `%s`\n", scope.Path, scope.Pattern)
		}
		files["references/catalog/"+name] = []byte(doc)
	}
	files["references/catalog/index.md"] = []byte(index)
	return files, nil
}
