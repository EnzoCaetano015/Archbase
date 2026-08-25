package workspace

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/EnzoCaetano015/Archbase/internal/contracts"
	archfs "github.com/EnzoCaetano015/Archbase/internal/filesystem"
	"github.com/EnzoCaetano015/Archbase/internal/patterns"
	"github.com/EnzoCaetano015/Archbase/internal/registry"
)

var (
	ErrScopeNotFound    = errors.New("Archbase scope not found")
	ErrInvalidScope     = errors.New("invalid Archbase scope")
	ErrPatternExists    = errors.New("local pattern already exists")
	ErrInvalidLocalRoot = errors.New("local pattern root must remain inside .archbase")
)

type FileSystem interface {
	Exists(string) (bool, error)
	EnsureDir(string) error
	WriteFileAtomic(string, []byte, bool) error
	Lstat(string) (fs.FileInfo, error)
	MkdirTemp(string, string) (string, error)
	Rename(string, string) error
	RemoveAll(string) error
	DirFS(string) fs.FS
}

type OSFileSystem struct{}

func (OSFileSystem) Exists(value string) (bool, error) { return archfs.Exists(value) }
func (OSFileSystem) EnsureDir(value string) error      { return archfs.EnsureDir(value) }
func (OSFileSystem) WriteFileAtomic(value string, data []byte, overwrite bool) error {
	return archfs.WriteFileAtomic(value, data, overwrite)
}
func (OSFileSystem) Lstat(value string) (fs.FileInfo, error) { return os.Lstat(value) }
func (OSFileSystem) MkdirTemp(dir, pattern string) (string, error) {
	return os.MkdirTemp(dir, pattern)
}
func (OSFileSystem) Rename(oldPath, newPath string) error { return os.Rename(oldPath, newPath) }
func (OSFileSystem) RemoveAll(value string) error         { return os.RemoveAll(value) }
func (OSFileSystem) DirFS(value string) fs.FS             { return os.DirFS(value) }

type Service struct {
	fs       FileSystem
	registry *registry.Resolver
}

func NewService(fsys FileSystem, resolver *registry.Resolver) (*Service, error) {
	if fsys == nil {
		return nil, errors.New("workspace filesystem is required")
	}
	if resolver == nil {
		return nil, errors.New("workspace registry resolver is required")
	}
	return &Service{fs: fsys, registry: resolver}, nil
}

type Installed struct {
	ScopeDirectory   string
	PatternDirectory string
	PatternID        string
	Warnings         []error
}

func (s *Service) Add(ctx context.Context, rawID, scopePath string) (Installed, error) {
	resolved, err := s.registry.Resolve(ctx, rawID)
	if err != nil {
		return Installed{}, err
	}
	id := resolved.Pattern.Entry.ID.String()
	origin := &contracts.Origin{
		Registry: resolved.Pattern.Source,
		ID:       id,
		Version:  resolved.Pattern.Bundle.Manifest.Version,
	}
	return s.install(scopePath, patternDirectoryName(id), resolved.Pattern.Bundle, origin, resolved.Warnings)
}

func (s *Service) Create(ctx context.Context, name, scopePath, from string) (Installed, error) {
	localID := "local/" + name + "@1"
	if _, err := registry.ParsePatternID(localID); err != nil {
		return Installed{}, fmt.Errorf("invalid local pattern name %q: %w", name, err)
	}
	if from == "" {
		manifest := contracts.Manifest{
			SchemaVersion: 1,
			ID:            localID,
			Name:          name,
			Description:   "Local customizable structural pattern.",
			Type:          "pattern",
			Version:       "1.0.0",
			Structure: contracts.PatternStructure{
				Root:  "{{Name}}",
				Files: []contracts.PatternFile{{Source: "Example.txt", Destination: "{{Name}}.txt", Required: true}},
			},
			AllowedChanges: []string{"identifiers", "content"},
			Preserve:       []string{"file-responsibility"},
		}
		manifestContent, err := contracts.EncodeManifest(manifest)
		if err != nil {
			return Installed{}, err
		}
		bundle := patterns.Bundle{
			Manifest:        manifest,
			ManifestContent: manifestContent,
			Files:           []patterns.File{{Spec: manifest.Structure.Files[0], Present: true, Content: []byte("Replace this example with the required structure.\n")}},
		}
		return s.install(scopePath, name, bundle, nil, nil)
	}

	resolved, err := s.registry.Resolve(ctx, from)
	if err != nil {
		return Installed{}, err
	}
	bundle := cloneBundle(resolved.Pattern.Bundle)
	originalID := bundle.Manifest.ID
	originalVersion := bundle.Manifest.Version
	bundle.Manifest.ID = localID
	bundle.Manifest.Name = name
	bundle.ManifestContent, err = contracts.EncodeManifest(bundle.Manifest)
	if err != nil {
		return Installed{}, err
	}
	origin := &contracts.Origin{Registry: resolved.Pattern.Source, ID: originalID, Version: originalVersion}
	return s.install(scopePath, name, bundle, origin, resolved.Warnings)
}

func cloneBundle(source patterns.Bundle) patterns.Bundle {
	result := source
	result.Manifest.AllowedChanges = append([]string(nil), source.Manifest.AllowedChanges...)
	result.Manifest.Preserve = append([]string(nil), source.Manifest.Preserve...)
	result.Manifest.Structure.Files = append([]contracts.PatternFile(nil), source.Manifest.Structure.Files...)
	result.Files = make([]patterns.File, len(source.Files))
	for index, file := range source.Files {
		result.Files[index] = file
		result.Files[index].Content = append([]byte(nil), file.Content...)
	}
	return result
}

func patternDirectoryName(rawID string) string {
	name := rawID[strings.LastIndex(rawID, "/")+1:]
	return strings.Replace(name, "@", "-", 1)
}

func (s *Service) install(scopePath, directoryName string, bundle patterns.Bundle, origin *contracts.Origin, warnings []error) (result Installed, returnedErr error) {
	scopeDirectory, err := filepath.Abs(scopePath)
	if err != nil {
		return result, fmt.Errorf("resolve scope path %q: %w", scopePath, err)
	}
	if err := s.fs.EnsureDir(scopeDirectory); err != nil {
		return result, err
	}
	info, err := s.fs.Lstat(scopeDirectory)
	if err != nil {
		return result, fmt.Errorf("inspect scope path %q: %w", scopeDirectory, err)
	}
	if info.Mode()&fs.ModeSymlink != 0 || !info.IsDir() {
		return result, fmt.Errorf("scope path %q must be a regular directory", scopeDirectory)
	}

	archbaseDirectory := filepath.Join(scopeDirectory, ".archbase")
	archbaseExisted, err := s.fs.Exists(archbaseDirectory)
	if err != nil {
		return result, err
	}
	if archbaseExisted {
		archbaseInfo, statErr := s.fs.Lstat(archbaseDirectory)
		if statErr != nil || archbaseInfo.Mode()&fs.ModeSymlink != 0 || !archbaseInfo.IsDir() {
			return result, fmt.Errorf("invalid .archbase directory %q", archbaseDirectory)
		}
		scopeFile := filepath.Join(archbaseDirectory, "scope.yaml")
		scopeInfo, statErr := s.fs.Lstat(scopeFile)
		if statErr != nil || scopeInfo.Mode()&fs.ModeSymlink != 0 || !scopeInfo.Mode().IsRegular() {
			return result, fmt.Errorf("%w at %q: scope file is missing, a symlink, or not regular", ErrInvalidScope, scopeFile)
		}
		if _, loadErr := contracts.LoadScope(scopeFile); loadErr != nil {
			return result, fmt.Errorf("%w at %q: %v", ErrInvalidScope, scopeFile, loadErr)
		}
	}
	patternsDirectory := filepath.Join(archbaseDirectory, "patterns")
	if err := s.fs.EnsureDir(patternsDirectory); err != nil {
		return result, err
	}
	if !archbaseExisted {
		defer func() {
			if returnedErr != nil {
				_ = s.fs.RemoveAll(archbaseDirectory)
			}
		}()
	}

	patternDirectory := filepath.Join(patternsDirectory, directoryName)
	if exists, checkErr := s.fs.Exists(patternDirectory); checkErr != nil {
		return result, checkErr
	} else if exists {
		return result, fmt.Errorf("%w: %s", ErrPatternExists, patternDirectory)
	}
	staging, err := s.fs.MkdirTemp(patternsDirectory, ".archbase-pattern-*")
	if err != nil {
		return result, fmt.Errorf("create pattern staging directory in %q: %w", patternsDirectory, err)
	}
	defer s.fs.RemoveAll(staging)
	manifestContent := bundle.ManifestContent
	if len(manifestContent) == 0 {
		manifestContent, err = contracts.EncodeManifest(bundle.Manifest)
		if err != nil {
			return result, err
		}
	}
	if err := s.fs.WriteFileAtomic(filepath.Join(staging, "manifest.yaml"), manifestContent, false); err != nil {
		return result, err
	}
	for _, file := range bundle.Files {
		if !file.Present {
			continue
		}
		destination := filepath.Join(staging, filepath.FromSlash(strings.ReplaceAll(file.Spec.Source, "\\", "/")))
		if err := s.fs.WriteFileAtomic(destination, file.Content, false); err != nil {
			return result, err
		}
	}
	if _, err := patterns.Load(s.fs.DirFS(staging), "."); err != nil {
		return result, fmt.Errorf("validate staged pattern %q: %w", staging, err)
	}
	if err := s.fs.Rename(staging, patternDirectory); err != nil {
		return result, fmt.Errorf("promote pattern to %q: %w", patternDirectory, err)
	}
	committed := true
	defer func() {
		if returnedErr != nil && committed {
			_ = s.fs.RemoveAll(patternDirectory)
		}
	}()

	scope := contracts.Scope{
		SchemaVersion: 1,
		Scope:         contracts.ScopeSelector{Path: "."},
		Pattern: contracts.ScopePattern{
			ID:     bundle.Manifest.ID,
			Source: "local",
			Root:   path.Join("patterns", directoryName),
		},
		Origin: origin,
		Behavior: contracts.ScopeBehavior{
			NearestScopeWins:        true,
			AllowLocalCustomization: true,
		},
	}
	scopeContent, err := contracts.EncodeScope(scope)
	if err != nil {
		return result, err
	}
	if err := s.fs.WriteFileAtomic(filepath.Join(archbaseDirectory, "scope.yaml"), scopeContent, archbaseExisted); err != nil {
		return result, err
	}
	committed = false
	return Installed{ScopeDirectory: scopeDirectory, PatternDirectory: patternDirectory, PatternID: bundle.Manifest.ID, Warnings: warnings}, nil
}

type Resolution struct {
	ScopeDirectory    string
	ArchbaseDirectory string
	PatternRoot       string
	Scope             contracts.Scope
	Pattern           registry.Pattern
	Stale             bool
	Warnings          []error
}

func (s *Service) Resolve(ctx context.Context, target string) (Resolution, error) {
	absolute, err := filepath.Abs(target)
	if err != nil {
		return Resolution{}, fmt.Errorf("resolve target %q: %w", target, err)
	}
	start := absolute
	if info, statErr := s.fs.Lstat(absolute); statErr == nil {
		if info.Mode()&fs.ModeSymlink != 0 {
			return Resolution{}, fmt.Errorf("target %q must not be a symbolic link", absolute)
		}
		if info.Mode().IsRegular() {
			start = filepath.Dir(start)
		} else if !info.IsDir() {
			return Resolution{}, fmt.Errorf("target %q is not a regular file or directory", absolute)
		}
	} else if !errors.Is(statErr, fs.ErrNotExist) {
		return Resolution{}, fmt.Errorf("inspect target %q: %w", absolute, statErr)
	}

	for candidate := start; ; candidate = filepath.Dir(candidate) {
		archbaseDirectory := filepath.Join(candidate, ".archbase")
		exists, checkErr := s.fs.Exists(archbaseDirectory)
		if checkErr != nil {
			return Resolution{}, checkErr
		}
		if exists {
			return s.resolveScope(ctx, candidate, archbaseDirectory)
		}
		parent := filepath.Dir(candidate)
		if parent == candidate {
			break
		}
	}
	return Resolution{}, fmt.Errorf("%w for %q", ErrScopeNotFound, target)
}

func (s *Service) resolveScope(ctx context.Context, scopeDirectory, archbaseDirectory string) (Resolution, error) {
	info, err := s.fs.Lstat(archbaseDirectory)
	if err != nil || info.Mode()&fs.ModeSymlink != 0 || !info.IsDir() {
		return Resolution{}, fmt.Errorf("%w: .archbase path %q is not a regular directory", ErrInvalidScope, archbaseDirectory)
	}
	scopeFile := filepath.Join(archbaseDirectory, "scope.yaml")
	scopeInfo, err := s.fs.Lstat(scopeFile)
	if err != nil || scopeInfo.Mode()&fs.ModeSymlink != 0 || !scopeInfo.Mode().IsRegular() {
		return Resolution{}, fmt.Errorf("%w: scope file %q is missing, a symlink, or not regular", ErrInvalidScope, scopeFile)
	}
	scope, err := contracts.LoadScope(scopeFile)
	if err != nil {
		return Resolution{}, fmt.Errorf("%w: %v", ErrInvalidScope, err)
	}
	if scope.Pattern.Source == "registry" {
		resolved, resolveErr := s.registry.Resolve(ctx, scope.Pattern.ID)
		if resolveErr != nil {
			return Resolution{}, resolveErr
		}
		return Resolution{ScopeDirectory: scopeDirectory, ArchbaseDirectory: archbaseDirectory, Scope: scope, Pattern: resolved.Pattern, Stale: resolved.Stale, Warnings: resolved.Warnings}, nil
	}
	root, err := safeLocalRoot(scope.Pattern.Root)
	if err != nil {
		return Resolution{}, fmt.Errorf("%w at %q: %v", ErrInvalidLocalRoot, scopeFile, err)
	}
	bundle, err := patterns.Load(s.fs.DirFS(archbaseDirectory), root)
	if err != nil {
		return Resolution{}, fmt.Errorf("load local pattern %q: %w", filepath.Join(archbaseDirectory, filepath.FromSlash(root)), err)
	}
	if bundle.Manifest.ID != scope.Pattern.ID {
		return Resolution{}, fmt.Errorf("scope pattern ID %q does not match local manifest ID %q at %q", scope.Pattern.ID, bundle.Manifest.ID, scopeFile)
	}
	id, err := registry.ParsePatternID(bundle.Manifest.ID)
	if err != nil {
		return Resolution{}, err
	}
	pattern := registry.Pattern{
		Entry:  registry.Entry{ID: id, Version: bundle.Manifest.Version, Path: root, Description: bundle.Manifest.Description},
		Bundle: bundle,
		Source: "local",
	}
	return Resolution{ScopeDirectory: scopeDirectory, ArchbaseDirectory: archbaseDirectory, PatternRoot: filepath.Join(archbaseDirectory, filepath.FromSlash(root)), Scope: scope, Pattern: pattern}, nil
}

func safeLocalRoot(value string) (string, error) {
	if value == "" || strings.Contains(value, "\\") || strings.HasPrefix(value, "/") || filepath.IsAbs(value) || filepath.VolumeName(value) != "" {
		return "", fmt.Errorf("%q is not a slash-separated relative path", value)
	}
	cleaned := path.Clean(value)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") || cleaned != value {
		return "", fmt.Errorf("%q escapes or is not normalized", value)
	}
	return cleaned, nil
}
