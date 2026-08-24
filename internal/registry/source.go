package registry

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/EnzoCaetano015/Archbase/internal/contracts"
	"github.com/goccy/go-yaml"
)

var ErrPatternNotFound = errors.New("pattern not found")

type Entry struct {
	ID          PatternID
	Version     string
	Path        string
	Description string
}

type Pattern struct {
	Entry    Entry
	Manifest contracts.Manifest
	Files    fs.FS
	Source   string
}

type Source interface {
	Name() string
	Lookup(PatternID) (Pattern, error)
	List() []Entry
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
	for position, raw := range index.Patterns {
		id, err := ParsePatternID(raw.ID)
		if err != nil {
			return nil, fmt.Errorf("registry %s: patterns[%d]: %w", name, position, err)
		}
		if _, exists := entries[id]; exists {
			return nil, fmt.Errorf("registry %s: duplicate pattern ID %q", name, id)
		}
		cleanPath, err := safeRegistryPath(raw.Path)
		if err != nil {
			return nil, fmt.Errorf("registry %s: pattern %q: %w", name, id, err)
		}
		manifest, err := contracts.LoadManifestFS(fsys, path.Join(cleanPath, "manifest.yaml"))
		if err != nil {
			return nil, fmt.Errorf("registry %s: pattern %q: %w", name, id, err)
		}
		if manifest.ID != id.String() || manifest.Version != raw.Version {
			return nil, fmt.Errorf("registry %s: pattern %q metadata does not match its manifest", name, id)
		}
		entries[id] = Entry{ID: id, Version: raw.Version, Path: cleanPath, Description: raw.Description}
	}
	return &catalogSource{name: name, fsys: fsys, entries: entries}, nil
}

func (s *catalogSource) Name() string { return s.name }

func (s *catalogSource) Lookup(id PatternID) (Pattern, error) {
	entry, ok := s.entries[id]
	if !ok {
		return Pattern{}, fmt.Errorf("registry %s: %w: %s", s.name, ErrPatternNotFound, id)
	}
	manifest, err := contracts.LoadManifestFS(s.fsys, path.Join(entry.Path, "manifest.yaml"))
	if err != nil {
		return Pattern{}, fmt.Errorf("registry %s: load %s: %w", s.name, id, err)
	}
	patternFS, err := fs.Sub(s.fsys, entry.Path)
	if err != nil {
		return Pattern{}, fmt.Errorf("registry %s: open files for %s: %w", s.name, id, err)
	}
	return Pattern{Entry: entry, Manifest: manifest, Files: patternFS, Source: s.name}, nil
}

func (s *catalogSource) List() []Entry {
	result := make([]Entry, 0, len(s.entries))
	for _, entry := range s.entries {
		result = append(result, entry)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
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
