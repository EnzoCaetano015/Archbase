package registry

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/EnzoCaetano015/Archbase/internal/patterns"
	"github.com/goccy/go-yaml"
)

var ErrPatternNotFound = errors.New("pattern not found")

type Entry struct {
	ID          PatternID
	Version     string
	Path        string
	Description string
	Source      string
}

type Pattern struct {
	Entry  Entry
	Bundle patterns.Bundle
	Source string
}

type LookupResult struct {
	Pattern Pattern
	Stale   bool
	Warning error
}

type ListResult struct {
	Entries []Entry
	Stale   bool
	Warning error
}

type Source interface {
	Name() string
	Lookup(context.Context, PatternID) (LookupResult, error)
	List(context.Context) (ListResult, error)
}

type indexDocument struct {
	SchemaVersion int          `yaml:"schemaVersion"`
	Patterns      []indexEntry `yaml:"patterns"`
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
	entries map[PatternID]Entry
}

func newCatalogSource(name string, fsys fs.FS) (*catalogSource, error) {
	data, err := fs.ReadFile(fsys, "index.yaml")
	if err != nil {
		return nil, fmt.Errorf("registry %s: read index.yaml: %w", name, err)
	}
	var index indexDocument
	if err := yaml.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("registry %s: parse index.yaml: %w", name, err)
	}
	if index.SchemaVersion != 1 {
		return nil, fmt.Errorf("registry %s: unsupported schemaVersion %d", name, index.SchemaVersion)
	}
	entries := make(map[PatternID]Entry, len(index.Patterns))
	previousID := PatternID("")
	for position, raw := range index.Patterns {
		id, err := ParsePatternID(raw.ID)
		if err != nil {
			return nil, fmt.Errorf("registry %s: patterns[%d]: %w", name, position, err)
		}
		if _, exists := entries[id]; exists {
			return nil, fmt.Errorf("registry %s: duplicate pattern ID %q", name, id)
		}
		if previousID != "" && id < previousID {
			return nil, fmt.Errorf("registry %s: pattern index must be sorted by ID: %q appears after %q", name, id, previousID)
		}
		previousID = id
		cleanPath, err := safeRegistryPath(raw.Path)
		if err != nil {
			return nil, fmt.Errorf("registry %s: pattern %q: %w", name, id, err)
		}
		bundle, err := patterns.Load(fsys, cleanPath)
		if err != nil {
			return nil, fmt.Errorf("registry %s: pattern %q: %w", name, id, err)
		}
		if bundle.Manifest.ID != id.String() || bundle.Manifest.Version != raw.Version {
			return nil, fmt.Errorf("registry %s: pattern %q metadata does not match its manifest", name, id)
		}
		entries[id] = Entry{ID: id, Version: raw.Version, Path: cleanPath, Description: raw.Description}
	}
	return &catalogSource{name: name, fsys: fsys, entries: entries}, nil
}

func (s *catalogSource) Name() string { return s.name }

func (s *catalogSource) Lookup(ctx context.Context, id PatternID) (LookupResult, error) {
	if err := ctx.Err(); err != nil {
		return LookupResult{}, err
	}
	entry, ok := s.entries[id]
	if !ok {
		return LookupResult{}, fmt.Errorf("registry %s: %w: %s", s.name, ErrPatternNotFound, id)
	}
	bundle, err := patterns.Load(s.fsys, entry.Path)
	if err != nil {
		return LookupResult{}, fmt.Errorf("registry %s: load %s: %w", s.name, id, err)
	}
	return LookupResult{Pattern: Pattern{Entry: entry, Bundle: bundle, Source: s.name}}, nil
}

func (s *catalogSource) List(ctx context.Context) (ListResult, error) {
	if err := ctx.Err(); err != nil {
		return ListResult{}, err
	}
	result := make([]Entry, 0, len(s.entries))
	for _, entry := range s.entries {
		result = append(result, entry)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return ListResult{Entries: result}, nil
}

func safeRegistryPath(value string) (string, error) {
	if value == "" || strings.Contains(value, "\\") || strings.HasPrefix(value, "/") {
		return "", fmt.Errorf("registry path must be a non-empty slash-separated relative path: %q", value)
	}
	cleaned := path.Clean(value)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") || cleaned != value {
		return "", fmt.Errorf("registry path escapes or is not normalized: %q", value)
	}
	return cleaned, nil
}
