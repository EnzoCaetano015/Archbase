package workspace

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

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
	CreateExclusive(string) error
}
type OSFileSystem struct{}

func (OSFileSystem) Exists(p string) (bool, error) { return archfs.Exists(p) }
func (OSFileSystem) EnsureDir(p string) error      { return archfs.EnsureDir(p) }
func (OSFileSystem) WriteFileAtomic(p string, b []byte, o bool) error {
	return archfs.WriteFileAtomic(p, b, o)
}
func (OSFileSystem) Lstat(p string) (fs.FileInfo, error)   { return os.Lstat(p) }
func (OSFileSystem) MkdirTemp(p, n string) (string, error) { return os.MkdirTemp(p, n) }
func (OSFileSystem) Rename(a, b string) error              { return os.Rename(a, b) }
func (OSFileSystem) RemoveAll(p string) error              { return os.RemoveAll(p) }
func (OSFileSystem) DirFS(p string) fs.FS                  { return os.DirFS(p) }
func (OSFileSystem) CreateExclusive(p string) error {
	f, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(p)
		return err
	}
	return nil
}

type Service struct {
	fs          FileSystem
	registry    *registry.Resolver
	projectRoot string
}

func NewService(fsys FileSystem, resolver *registry.Resolver) (*Service, error) {
	if fsys == nil || resolver == nil {
		return nil, errors.New("workspace filesystem and registry resolver are required")
	}
	return &Service{fs: fsys, registry: resolver}, nil
}
func (s *Service) WithProjectRoot(root string) *Service { c := *s; c.projectRoot = root; return &c }

type Installed struct {
	ScopeDirectory, PatternDirectory, PatternID string
	Warnings                                    []error
}
type Resolution struct {
	ScopeDirectory, ArchbaseDirectory, PatternRoot string
	Scope                                          contracts.Scope
	Pattern                                        registry.Pattern
	Stale                                          bool
	Warnings                                       []error
}

func (s *Service) Add(ctx context.Context, id, target string) (Installed, error) {
	resolved, err := s.registry.Resolve(ctx, id)
	if err != nil {
		return Installed{}, err
	}
	origin := &contracts.Origin{Registry: resolved.Pattern.Source, ID: resolved.Pattern.Entry.ID.String(), Version: resolved.Pattern.Bundle.Manifest.Version}
	return s.install(ctx, target, resolved.Pattern.Bundle, origin, resolved.Warnings)
}
func (s *Service) Create(ctx context.Context, name, target, from string) (Installed, error) {
	id := "local/" + name + "@1"
	if _, err := registry.ParsePatternID(id); err != nil {
		return Installed{}, fmt.Errorf("invalid local pattern name %q: %w", name, err)
	}
	var bundle patterns.Bundle
	var origin *contracts.Origin
	var warnings []error
	if from != "" {
		resolved, err := s.registry.Resolve(ctx, from)
		if err != nil {
			return Installed{}, err
		}
		bundle = resolved.Pattern.Bundle
		origin = &contracts.Origin{Registry: resolved.Pattern.Source, ID: bundle.Manifest.ID, Version: bundle.Manifest.Version}
		warnings = resolved.Warnings
	} else {
		spec := contracts.PatternFile{Source: "Example.txt", Destination: "{{Name}}.txt", Required: true}
		bundle = patterns.Bundle{Manifest: contracts.Manifest{SchemaVersion: 1, Type: "pattern", Version: "1.0.0", Description: "Local customizable structural pattern.", Structure: contracts.PatternStructure{Root: "{{Name}}", Files: []contracts.PatternFile{spec}}, AllowedChanges: []string{"identifiers", "content"}, Preserve: []string{"file-responsibility"}}, Files: []patterns.File{{Spec: spec, Present: true, Content: []byte("Replace this example with the required structure.\n")}}}
	}
	bundle.Manifest.ID, bundle.Manifest.Name = id, name
	return s.install(ctx, target, bundle, origin, warnings)
}
func safeLocalRoot(value string) (string, error) {
	if value == "" || strings.ContainsAny(value, "\\:") || strings.HasPrefix(value, "/") || path.Clean(value) != value || value == ".." || strings.HasPrefix(value, "../") || value == "." {
		return "", fmt.Errorf("%q escapes or is not normalized", value)
	}
	return value, nil
}
func (s *Service) checkPath(p string) error {
	absolute, err := filepath.Abs(p)
	if err != nil {
		return err
	}
	boundary := s.projectRoot
	if boundary == "" {
		boundary, _ = os.Getwd()
	}
	boundary, _ = filepath.Abs(boundary)
	for current := absolute; ; current = filepath.Dir(current) {
		info, err := s.fs.Lstat(current)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("inspect path %q: %w", current, err)
		}
		if err == nil && info.Mode()&fs.ModeSymlink != 0 {
			return fmt.Errorf("symbolic links are not allowed at %q", current)
		}
		if err == nil && current != absolute && !info.IsDir() {
			return fmt.Errorf("path component %q must be a directory", current)
		}
		if current == boundary || current == filepath.Dir(current) {
			break
		}
	}
	return nil
}
func relative(root, target string) (string, error) {
	p, err := filepath.Rel(root, target)
	if err != nil || p == ".." || strings.HasPrefix(p, ".."+string(filepath.Separator)) || filepath.IsAbs(p) {
		return "", fmt.Errorf("path %q escapes project root %q", target, root)
	}
	return filepath.ToSlash(p), nil
}
func (s *Service) rootForWrite() (string, error) {
	root := s.projectRoot
	if root == "" {
		var err error
		root, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if err = s.WithProjectRoot(root).checkPath(root); err != nil {
		return "", err
	}
	info, err := s.fs.Lstat(root)
	if err != nil {
		return "", fmt.Errorf("inspect project root %q: %w", root, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("project root %q must be a regular directory", root)
	}
	return root, nil
}
func (s *Service) load(root string) (contracts.Workspace, error) {
	s = s.WithProjectRoot(root)
	dir := filepath.Join(root, ".archbase")
	if err := s.checkPath(dir); err != nil {
		return contracts.Workspace{}, err
	}
	if err := s.checkPath(filepath.Join(dir, "scopes.yaml")); err != nil {
		return contracts.Workspace{}, err
	}
	value, err := contracts.LoadWorkspaceFS(s.fs.DirFS(dir), "scopes.yaml")
	if err != nil {
		return value, fmt.Errorf("%w at %q: %w", ErrInvalidScope, dir, err)
	}
	return value, nil
}

// The owner removes the lock on normal exit. A crashed writer requires manual lock removal.
func (s *Service) lock(ctx context.Context, root string) (func(), error) {
	lock := filepath.Join(root, ".archbase.lock")
	timer := time.NewTicker(20 * time.Millisecond)
	defer timer.Stop()
	permissionRetries := 0
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		err := s.fs.CreateExclusive(lock)
		if err == nil {
			return func() { _ = s.fs.RemoveAll(lock) }, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			// Windows may briefly report ACCESS_DENIED while the previous lock
			// is delete-pending. Retry that race, but do not hide persistent ACL errors.
			if runtime.GOOS == "windows" && errors.Is(err, fs.ErrPermission) && permissionRetries < 10 {
				permissionRetries++
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-timer.C:
				}
				continue
			}
			return nil, fmt.Errorf("lock workspace %q: %w", lock, err)
		}
		permissionRetries = 0
		if err := s.checkPath(lock); err != nil {
			return nil, err
		}
		if info, err := s.fs.Lstat(lock); err == nil && !info.Mode().IsRegular() {
			return nil, fmt.Errorf("workspace lock %q must be a regular file", lock)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}
func (s *Service) install(ctx context.Context, target string, bundle patterns.Bundle, origin *contracts.Origin, warnings []error) (result Installed, returnedErr error) {
	if err := rejectTraversal(target); err != nil {
		return result, err
	}
	root, err := s.rootForWrite()
	if err != nil {
		return result, err
	}
	s = s.WithProjectRoot(root)
	absolute, err := filepath.Abs(target)
	if err != nil {
		return result, err
	}
	scopePath, err := relative(root, absolute)
	if err != nil {
		return result, err
	}
	if scopePath == ".archbase" || strings.HasPrefix(scopePath, ".archbase/") {
		return result, fmt.Errorf("scope %q is inside workspace storage", target)
	}
	if err = s.checkPath(absolute); err != nil {
		return result, err
	}
	if info, e := s.fs.Lstat(absolute); e == nil && !info.IsDir() {
		return result, fmt.Errorf("scope %q must be a directory", absolute)
	} else if e != nil && !errors.Is(e, fs.ErrNotExist) {
		return result, e
	}
	bundle, err = patterns.DocumentBundle(bundle)
	if err != nil {
		return result, err
	}
	unlock, err := s.lock(ctx, root)
	if err != nil {
		return result, err
	}
	defer unlock()
	dir := filepath.Join(root, ".archbase")
	if err = s.checkPath(dir); err != nil {
		return result, err
	}
	existed, err := s.fs.Exists(dir)
	if err != nil {
		return result, err
	}
	index := contracts.Workspace{SchemaVersion: 1, Scopes: []contracts.Scope{}}
	if existed {
		index, err = s.load(root)
		if err != nil {
			return result, err
		}
	}
	if err = s.fs.EnsureDir(absolute); err != nil {
		return result, err
	}
	key := fmt.Sprintf("patterns/%x/%x", sha256.Sum256([]byte(scopePath)), sha256.Sum256([]byte(bundle.Manifest.ID)))
	final := filepath.Join(dir, filepath.FromSlash(key))
	if err = s.checkPath(final); err != nil {
		return result, err
	}
	if exists, e := s.fs.Exists(final); e != nil {
		return result, e
	} else if exists {
		return result, fmt.Errorf("%w: %s", ErrPatternExists, final)
	}
	stage, err := s.fs.MkdirTemp(root, ".archbase-stage-*")
	if err != nil {
		return result, err
	}
	defer s.fs.RemoveAll(stage)
	stageBundle := filepath.Join(stage, filepath.FromSlash(key))
	if err = s.fs.EnsureDir(stageBundle); err != nil {
		return result, err
	}
	if err = s.fs.WriteFileAtomic(filepath.Join(stageBundle, "manifest.yaml"), bundle.ManifestContent, false); err != nil {
		return result, err
	}
	for _, f := range bundle.Files {
		if f.Present {
			content := f.DocumentContent
			if content == nil {
				content = f.Content
			}
			if err = s.fs.WriteFileAtomic(filepath.Join(stageBundle, filepath.FromSlash(f.Spec.Source)), content, false); err != nil {
				return result, err
			}
		}
	}
	if _, err = patterns.Load(s.fs.DirFS(stageBundle), "."); err != nil {
		return result, err
	}
	scope := contracts.Scope{SchemaVersion: 1, Scope: contracts.ScopeSelector{Path: scopePath}, Pattern: contracts.ScopePattern{ID: bundle.Manifest.ID, Source: "local", Root: key}, Origin: origin, Behavior: contracts.ScopeBehavior{NearestScopeWins: true, AllowLocalCustomization: true}}
	found := false
	for i := range index.Scopes {
		if index.Scopes[i].Scope.Path == scopePath {
			scope.Metadata = index.Scopes[i].Metadata
			index.Scopes[i] = scope
			found = true
			break
		}
	}
	if !found {
		index.Scopes = append(index.Scopes, scope)
	}
	sort.Slice(index.Scopes, func(i, j int) bool { return index.Scopes[i].Scope.Path < index.Scopes[j].Scope.Path })
	data, err := contracts.EncodeWorkspace(index)
	if err != nil {
		return result, err
	}
	if err = ctx.Err(); err != nil {
		return result, err
	}
	if !existed {
		if err = s.fs.WriteFileAtomic(filepath.Join(stage, "scopes.yaml"), data, false); err != nil {
			return result, err
		}
		if err = s.fs.Rename(stage, dir); err != nil {
			return result, fmt.Errorf("promote workspace %q: %w", dir, err)
		}
	} else {
		if err = s.fs.EnsureDir(filepath.Dir(final)); err != nil {
			return result, err
		}
		if err = s.fs.Rename(stageBundle, final); err != nil {
			return result, err
		}
		defer func() {
			if returnedErr != nil {
				if err := s.fs.RemoveAll(final); err != nil {
					returnedErr = errors.Join(returnedErr, fmt.Errorf("rollback pattern %q: %w", final, err))
				}
			}
		}()
		previous, readErr := fs.ReadFile(s.fs.DirFS(dir), "scopes.yaml")
		if readErr != nil {
			return result, readErr
		}
		if err = s.fs.WriteFileAtomic(filepath.Join(dir, "scopes.yaml"), data, true); err != nil {
			// Restore only if the writer actually changed the index before failing.
			current, readErr := fs.ReadFile(s.fs.DirFS(dir), "scopes.yaml")
			if readErr != nil || string(current) != string(previous) {
				err = errors.Join(err, s.fs.WriteFileAtomic(filepath.Join(dir, "scopes.yaml"), previous, true))
			}
			return result, err
		}
	}
	return Installed{ScopeDirectory: absolute, PatternDirectory: final, PatternID: bundle.Manifest.ID, Warnings: warnings}, nil
}
func (s *Service) Resolve(ctx context.Context, target string) (Resolution, error) {
	if err := rejectTraversal(target); err != nil {
		return Resolution{}, err
	}
	if err := ctx.Err(); err != nil {
		return Resolution{}, err
	}
	absolute, err := filepath.Abs(target)
	if err != nil {
		return Resolution{}, err
	}
	start := absolute
	if info, e := s.fs.Lstat(start); e == nil {
		if info.Mode().IsRegular() {
			start = filepath.Dir(start)
		} else if !info.IsDir() {
			return Resolution{}, fmt.Errorf("target %q is not a regular file or directory", start)
		}
	} else if !errors.Is(e, fs.ErrNotExist) {
		return Resolution{}, e
	}
	root := s.projectRoot
	if root != "" {
		root, err = filepath.Abs(root)
		if err != nil {
			return Resolution{}, err
		}
		if _, err = relative(root, absolute); err != nil {
			return Resolution{}, err
		}
	} else {
		for candidate := start; ; candidate = filepath.Dir(candidate) {
			exists, e := s.fs.Exists(filepath.Join(candidate, ".archbase"))
			if e != nil {
				return Resolution{}, e
			}
			if exists {
				root = candidate
				break
			}
			if candidate == filepath.Dir(candidate) {
				break
			}
		}
	}
	if root == "" {
		return Resolution{}, fmt.Errorf("%w for %q", ErrScopeNotFound, target)
	}
	s = s.WithProjectRoot(root)
	if err = s.checkPath(absolute); err != nil {
		return Resolution{}, err
	}
	index, err := s.load(root)
	if err != nil {
		return Resolution{}, err
	}
	rel, err := relative(root, absolute)
	if err != nil {
		return Resolution{}, err
	}
	var chosen *contracts.Scope
	for i := range index.Scopes {
		scope := &index.Scopes[i]
		p := scope.Scope.Path
		if (p == "." || rel == p || strings.HasPrefix(rel, p+"/")) && (chosen == nil || len(p) > len(chosen.Scope.Path)) {
			chosen = scope
		}
	}
	if chosen == nil {
		return Resolution{}, fmt.Errorf("%w for %q", ErrScopeNotFound, target)
	}
	return s.resolveScope(ctx, root, *chosen)
}
func (s *Service) resolveScope(ctx context.Context, root string, scope contracts.Scope) (Resolution, error) {
	s = s.WithProjectRoot(root)
	if err := ctx.Err(); err != nil {
		return Resolution{}, err
	}
	dir := filepath.Join(root, ".archbase")
	result := Resolution{ScopeDirectory: filepath.Join(root, filepath.FromSlash(scope.Scope.Path)), ArchbaseDirectory: dir, Scope: scope}
	if err := s.checkPath(result.ScopeDirectory); err != nil {
		return Resolution{}, err
	}
	if info, err := s.fs.Lstat(result.ScopeDirectory); err == nil && !info.IsDir() {
		return Resolution{}, fmt.Errorf("scope %q must be a directory", result.ScopeDirectory)
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Resolution{}, err
	}
	if scope.Pattern.Source == "registry" {
		r, err := s.registry.Resolve(ctx, scope.Pattern.ID)
		result.Pattern, result.Stale, result.Warnings = r.Pattern, r.Stale, r.Warnings
		return result, err
	}
	local, err := safeLocalRoot(scope.Pattern.Root)
	if err != nil {
		return Resolution{}, fmt.Errorf("%w at %q: %w", ErrInvalidLocalRoot, filepath.Join(dir, "scopes.yaml"), err)
	}
	result.PatternRoot = filepath.Join(dir, filepath.FromSlash(local))
	if err = s.checkPath(result.PatternRoot); err != nil {
		return Resolution{}, err
	}
	bundle, err := patterns.Load(s.fs.DirFS(dir), local)
	if err != nil {
		return Resolution{}, fmt.Errorf("load local pattern %q: %w", result.PatternRoot, err)
	}
	if bundle.Manifest.ID != scope.Pattern.ID {
		return Resolution{}, fmt.Errorf("scope pattern ID %q does not match local manifest ID %q at %q", scope.Pattern.ID, bundle.Manifest.ID, result.PatternRoot)
	}
	id, err := registry.ParsePatternID(bundle.Manifest.ID)
	if err != nil {
		return Resolution{}, err
	}
	result.Pattern = registry.Pattern{Entry: registry.Entry{ID: id, Version: bundle.Manifest.Version, Path: local, Description: bundle.Manifest.Description}, Bundle: bundle, Source: "local"}
	return result, nil
}

func rejectTraversal(value string) error {
	for _, part := range strings.Split(strings.ReplaceAll(value, "\\", "/"), "/") {
		if part == ".." {
			return fmt.Errorf("traversal is not allowed in path %q", value)
		}
	}
	return nil
}
func (s *Service) List(ctx context.Context, root string) ([]Resolution, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	index, err := s.load(root)
	if err != nil {
		return nil, err
	}
	result := make([]Resolution, 0, len(index.Scopes))
	for _, scope := range index.Scopes {
		r, err := s.resolveScope(ctx, root, scope)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, nil
}
