package rules

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/EnzoCaetano015/Archbase/internal/contracts"
	"github.com/goccy/go-yaml"
)

type indexDocument struct {
	SchemaVersion int          `yaml:"schemaVersion"`
	Rules         []indexEntry `yaml:"rules"`
}

type indexEntry struct {
	ID          string `yaml:"id"`
	Version     string `yaml:"version"`
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
}

type catalogSource struct {
	name    string
	fsys    fs.FS
	entries map[RuleID]Entry
}

func newCatalogSource(name string, fsys fs.FS) (*catalogSource, error) {
	if err := rejectSymlinkComponents(fsys, "index.yaml"); err != nil {
		return nil, fmt.Errorf("rule registry %s: inspect index.yaml: %w", name, err)
	}
	data, err := fs.ReadFile(fsys, "index.yaml")
	if err != nil {
		return nil, fmt.Errorf("rule registry %s: read index.yaml: %w", name, err)
	}
	var index indexDocument
	if err := yaml.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("rule registry %s: parse index.yaml: %w", name, err)
	}
	if index.SchemaVersion != 1 {
		return nil, fmt.Errorf("rule registry %s: unsupported schemaVersion %d", name, index.SchemaVersion)
	}
	entries := make(map[RuleID]Entry, len(index.Rules))
	previousID := RuleID("")
	for position, raw := range index.Rules {
		id, err := ParseRuleID(raw.ID)
		if err != nil {
			return nil, fmt.Errorf("rule registry %s: rules[%d]: %w", name, position, err)
		}
		if _, exists := entries[id]; exists {
			return nil, fmt.Errorf("rule registry %s: duplicate rule ID %q", name, id)
		}
		if previousID != "" && id < previousID {
			return nil, fmt.Errorf("rule registry %s: index must be sorted by ID: %q appears after %q", name, id, previousID)
		}
		previousID = id
		cleanPath, err := safeCatalogPath(raw.Path)
		if err != nil {
			return nil, fmt.Errorf("rule registry %s: rule %q: %w", name, id, err)
		}
		rulePath := path.Join(cleanPath, "rule.yaml")
		if err := rejectSymlinkComponents(fsys, rulePath); err != nil {
			return nil, fmt.Errorf("rule registry %s: rule %q at %q: %w", name, id, rulePath, err)
		}
		info, err := fs.Stat(fsys, rulePath)
		if err != nil || !info.Mode().IsRegular() {
			if err == nil {
				err = errors.New("rule document is not a regular file")
			}
			return nil, fmt.Errorf("rule registry %s: rule %q at %q: %w", name, id, rulePath, err)
		}
		definition, err := contracts.LoadRuleFS(fsys, rulePath)
		if err != nil {
			return nil, fmt.Errorf("rule registry %s: rule %q: %w", name, id, err)
		}
		if definition.ID != id.String() || definition.Version != raw.Version {
			return nil, fmt.Errorf("rule registry %s: rule %q metadata does not match rule.yaml", name, id)
		}
		entries[id] = Entry{ID: id, Version: raw.Version, Path: cleanPath, Description: raw.Description, Source: name}
	}
	return &catalogSource{name: name, fsys: fsys, entries: entries}, nil
}

func (s *catalogSource) Name() string { return s.name }

func (s *catalogSource) Lookup(ctx context.Context, id RuleID) (LookupResult, error) {
	if err := ctx.Err(); err != nil {
		return LookupResult{}, err
	}
	entry, exists := s.entries[id]
	if !exists {
		return LookupResult{}, fmt.Errorf("rule registry %s: %w: %s", s.name, ErrRuleNotFound, id)
	}
	rulePath := path.Join(entry.Path, "rule.yaml")
	if err := rejectSymlinkComponents(s.fsys, rulePath); err != nil {
		return LookupResult{}, fmt.Errorf("rule registry %s: inspect %s: %w", s.name, id, err)
	}
	info, err := fs.Stat(s.fsys, rulePath)
	if err != nil || !info.Mode().IsRegular() {
		if err == nil {
			err = errors.New("rule document is not a regular file")
		}
		return LookupResult{}, fmt.Errorf("rule registry %s: inspect %s: %w", s.name, id, err)
	}
	definition, err := contracts.LoadRuleFS(s.fsys, rulePath)
	if err != nil {
		return LookupResult{}, fmt.Errorf("rule registry %s: load %s: %w", s.name, id, err)
	}
	content, err := fs.ReadFile(s.fsys, rulePath)
	if err != nil {
		return LookupResult{}, fmt.Errorf("rule registry %s: read %s: %w", s.name, id, err)
	}
	if definition.ID != entry.ID.String() || definition.Version != entry.Version {
		return LookupResult{}, fmt.Errorf("rule registry %s: rule %q metadata does not match rule.yaml", s.name, id)
	}
	return LookupResult{Rule: Rule{Entry: entry, Definition: definition, Content: content, Source: s.name}}, nil
}

func (s *catalogSource) List(ctx context.Context) (SourceListResult, error) {
	if err := ctx.Err(); err != nil {
		return SourceListResult{}, err
	}
	entries := make([]Entry, 0, len(s.entries))
	for _, entry := range s.entries {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
	return SourceListResult{Entries: entries}, nil
}

func safeCatalogPath(value string) (string, error) {
	if value == "" || strings.Contains(value, "\\") || strings.HasPrefix(value, "/") {
		return "", fmt.Errorf("catalog path must be a non-empty slash-separated relative path: %q", value)
	}
	cleaned := path.Clean(value)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") || cleaned != value {
		return "", fmt.Errorf("catalog path escapes or is not normalized: %q", value)
	}
	return cleaned, nil
}

func rejectSymlinkComponents(fsys fs.FS, filePath string) error {
	parts := strings.Split(filePath, "/")
	current := "."
	for _, part := range parts {
		entries, err := fs.ReadDir(fsys, current)
		if err != nil {
			return err
		}
		found := false
		for _, entry := range entries {
			if entry.Name() != part {
				continue
			}
			found = true
			if entry.Type()&fs.ModeSymlink != 0 {
				return errors.New("symbolic links are not allowed in rule catalogs")
			}
			break
		}
		if !found {
			return fs.ErrNotExist
		}
		current = path.Join(current, part)
	}
	return nil
}
