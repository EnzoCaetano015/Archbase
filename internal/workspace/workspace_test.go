package workspace

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/EnzoCaetano015/Archbase/internal/contracts"
	"github.com/EnzoCaetano015/Archbase/internal/registry"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func testService(t *testing.T, fsys FileSystem) *Service {
	t.Helper()
	embedded, err := registry.NewEmbeddedSource()
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := registry.NewResolver(embedded)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(fsys, resolver)
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func TestAddInstallsEmbeddedBundleAndScope(t *testing.T) {
	root := t.TempDir()
	service := testService(t, OSFileSystem{})
	installed, err := service.Add(context.Background(), "next/page@1234", filepath.Join(root, "src", "pages"))
	if err != nil {
		t.Fatal(err)
	}
	if installed.PatternID != "next/page@1234" || filepath.Base(installed.PatternDirectory) != "page-1234" {
		t.Fatalf("unexpected installation: %#v", installed)
	}
	for _, relative := range []string{"manifest.yaml", "Example/Example.tsx", "Example/Example.hook.ts", "Example/Example.utils.ts"} {
		if _, err := os.Stat(filepath.Join(installed.PatternDirectory, filepath.FromSlash(relative))); err != nil {
			t.Fatalf("expected installed file %s: %v", relative, err)
		}
	}
	if _, err := os.Stat(filepath.Join(installed.PatternDirectory, "README.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("README must not be part of the installed bundle: %v", err)
	}
	scope, err := contracts.LoadScope(filepath.Join(installed.ScopeDirectory, ".archbase", "scope.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if scope.Scope.Path != "." || scope.Pattern.Root != "patterns/page-1234" || scope.Origin == nil || scope.Origin.Registry != "official-embedded" {
		t.Fatalf("unexpected scope: %#v", scope)
	}
}

func TestAddInstallsFromGitRegistry(t *testing.T) {
	remote := createWorkspaceGitRegistry(t)
	gitSource, err := registry.NewGitSource(registry.GitSourceConfig{
		URL: fileRegistryURL(remote), Ref: "main", CacheRoot: t.TempDir(), TTL: registry.DefaultGitCacheTTL,
	})
	if err != nil {
		t.Fatal(err)
	}
	embedded, err := registry.NewEmbeddedSource()
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := registry.NewResolver(gitSource, embedded)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(OSFileSystem{}, resolver)
	if err != nil {
		t.Fatal(err)
	}
	installed, err := service.Add(context.Background(), "custom/widget@1", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(installed.PatternDirectory, "Example.txt"))
	if err != nil || string(content) != "from git\n" {
		t.Fatalf("unexpected installed Git content %q: %v", content, err)
	}
	scope, err := contracts.LoadScope(filepath.Join(installed.ScopeDirectory, ".archbase", "scope.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if scope.Origin == nil || !strings.HasPrefix(scope.Origin.Registry, "git:") {
		t.Fatalf("Git origin was not recorded: %#v", scope.Origin)
	}
}

func TestCreatePreservesStoredPatternsAndActivatesNewOne(t *testing.T) {
	root := t.TempDir()
	service := testService(t, OSFileSystem{})
	first, err := service.Add(context.Background(), "next/page@1234", root)
	if err != nil {
		t.Fatal(err)
	}
	created, err := service.Create(context.Background(), "pages-standard", root, "next/component@4821")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(first.PatternDirectory); err != nil {
		t.Fatalf("previous pattern was not preserved: %v", err)
	}
	manifest, err := contracts.LoadManifest(filepath.Join(created.PatternDirectory, "manifest.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if manifest.ID != "local/pages-standard@1" || manifest.Name != "pages-standard" {
		t.Fatalf("unexpected derived manifest: %#v", manifest)
	}
	scope, err := contracts.LoadScope(filepath.Join(root, ".archbase", "scope.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if scope.Pattern.ID != manifest.ID || scope.Origin == nil || scope.Origin.ID != "next/component@4821" {
		t.Fatalf("unexpected active scope: %#v", scope)
	}
	if _, err := service.Create(context.Background(), "pages-standard", root, ""); !errors.Is(err, ErrPatternExists) {
		t.Fatalf("expected collision, got %v", err)
	}
}

func TestCreateMinimalPatternAndRejectsInvalidName(t *testing.T) {
	service := testService(t, OSFileSystem{})
	root := t.TempDir()
	installed, err := service.Create(context.Background(), "pages-standard", root, "")
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := contracts.LoadManifest(filepath.Join(installed.PatternDirectory, "manifest.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if manifest.ID != "local/pages-standard@1" || manifest.Version != "1.0.0" {
		t.Fatalf("unexpected manifest: %#v", manifest)
	}
	if _, err := os.Stat(filepath.Join(installed.PatternDirectory, "Example.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create(context.Background(), "Pages Standard", filepath.Join(root, "other"), ""); err == nil || !strings.Contains(err.Error(), "invalid local pattern name") {
		t.Fatalf("expected strict name validation, got %v", err)
	}
}

func TestResolveUsesNearestScopeForFilesDirectoriesAndMissingPaths(t *testing.T) {
	service := testService(t, OSFileSystem{})
	root := t.TempDir()
	if _, err := service.Create(context.Background(), "parent", root, ""); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "src", "nested")
	if _, err := service.Create(context.Background(), "child", nested, ""); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(nested, "file.txt")
	if err := os.WriteFile(file, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{nested, file, filepath.Join(nested, "future", "deeper")} {
		resolved, err := service.Resolve(context.Background(), target)
		if err != nil {
			t.Fatalf("resolve %s: %v", target, err)
		}
		if resolved.Pattern.Entry.ID != "local/child@1" || resolved.ScopeDirectory != nested {
			t.Fatalf("nearest scope did not win for %s: %#v", target, resolved)
		}
	}
}

func TestResolveRegistryScopeAndRejectsCorruption(t *testing.T) {
	service := testService(t, OSFileSystem{})
	root := t.TempDir()
	archbase := filepath.Join(root, ".archbase")
	if err := os.MkdirAll(archbase, 0o755); err != nil {
		t.Fatal(err)
	}
	scope := contracts.Scope{SchemaVersion: 1, Scope: contracts.ScopeSelector{Path: "."}, Pattern: contracts.ScopePattern{ID: "next/hook@9214", Source: "registry"}, Behavior: contracts.ScopeBehavior{NearestScopeWins: true}}
	data, err := contracts.EncodeScope(scope)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(archbase, "scope.yaml"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	resolved, err := service.Resolve(context.Background(), root)
	if err != nil || resolved.Pattern.Entry.ID != "next/hook@9214" {
		t.Fatalf("registry scope resolution failed: %#v, %v", resolved, err)
	}
	if err := os.WriteFile(filepath.Join(archbase, "scope.yaml"), []byte("invalid: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Resolve(context.Background(), root); !errors.Is(err, ErrInvalidScope) {
		t.Fatalf("expected invalid scope error, got %v", err)
	}
}

func TestResolveRejectsEscapingRootAndManifestMismatch(t *testing.T) {
	service := testService(t, OSFileSystem{})
	root := t.TempDir()
	installed, err := service.Create(context.Background(), "example", root, "")
	if err != nil {
		t.Fatal(err)
	}
	scopePath := filepath.Join(root, ".archbase", "scope.yaml")
	scope, err := contracts.LoadScope(scopePath)
	if err != nil {
		t.Fatal(err)
	}
	scope.Pattern.Root = "../outside"
	data, err := contracts.EncodeScope(scope)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(scopePath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Resolve(context.Background(), root); !errors.Is(err, ErrInvalidLocalRoot) {
		t.Fatalf("expected confined-root error, got %v", err)
	}
	scope.Pattern.Root = "patterns/example"
	scope.Pattern.ID = "local/other@1"
	data, _ = contracts.EncodeScope(scope)
	_ = os.WriteFile(scopePath, data, 0o644)
	if _, err := service.Resolve(context.Background(), root); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("expected manifest mismatch, got %v", err)
	}
	_ = installed
}

func TestResolveRevalidatesRequiredLocalFiles(t *testing.T) {
	service := testService(t, OSFileSystem{})
	root := t.TempDir()
	installed, err := service.Create(context.Background(), "example", root, "")
	if err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(installed.PatternDirectory, "Example.txt")
	if err := os.Remove(missing); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Resolve(context.Background(), root); err == nil || !strings.Contains(err.Error(), missing) && !strings.Contains(err.Error(), "Example.txt") {
		t.Fatalf("expected required-file error with path, got %v", err)
	}
}

func TestResolveRejectsSymlinkedLocalRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires privileges on many Windows environments")
	}
	service := testService(t, OSFileSystem{})
	root := t.TempDir()
	if _, err := service.Create(context.Background(), "example", root, ""); err != nil {
		t.Fatal(err)
	}
	patternRoot := filepath.Join(root, ".archbase", "patterns", "example")
	realRoot := filepath.Join(root, "real")
	if err := os.Rename(patternRoot, realRoot); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realRoot, patternRoot); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Resolve(context.Background(), root); err == nil || !strings.Contains(err.Error(), "symbolic links") {
		t.Fatalf("expected symlink error, got %v", err)
	}
}

type scopeWriteFailureFS struct{ OSFileSystem }

func (scopeWriteFailureFS) WriteFileAtomic(value string, data []byte, overwrite bool) error {
	if filepath.Base(value) == "scope.yaml" && overwrite {
		return errors.New("simulated scope write failure")
	}
	return archWrite(OSFileSystem{}, value, data, overwrite)
}

func archWrite(fsys OSFileSystem, value string, data []byte, overwrite bool) error {
	return fsys.WriteFileAtomic(value, data, overwrite)
}

func TestFailedScopeUpdateRollsBackNewPattern(t *testing.T) {
	root := t.TempDir()
	initial := testService(t, OSFileSystem{})
	if _, err := initial.Create(context.Background(), "initial", root, ""); err != nil {
		t.Fatal(err)
	}
	service := testService(t, scopeWriteFailureFS{})
	if _, err := service.Add(context.Background(), "next/hook@9214", root); err == nil || !strings.Contains(err.Error(), "simulated") {
		t.Fatalf("expected simulated failure, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".archbase", "patterns", "hook-9214")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("new pattern was not rolled back: %v", err)
	}
	resolved, err := initial.Resolve(context.Background(), root)
	if err != nil || resolved.Pattern.Entry.ID != "local/initial@1" {
		t.Fatalf("previous scope was not preserved: %#v, %v", resolved, err)
	}
}

func createWorkspaceGitRegistry(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "remote")
	repository, err := git.PlainInit(root, false)
	if err != nil {
		t.Fatal(err)
	}
	write := func(relative, content string) {
		file := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("index.yaml", "schemaVersion: 1\npatterns:\n  - id: custom/widget@1\n    version: 1.0.0\n    path: custom/widget/1\n")
	write("custom/widget/1/manifest.yaml", `schemaVersion: 1
id: custom/widget@1
name: Git widget
type: pattern
version: 1.0.0
structure:
  root: "{{Name}}"
  files:
    - source: Example.txt
      destination: "{{Name}}.txt"
      required: true
allowedChanges:
  - content
preserve:
  - file-responsibility
`)
	write("custom/widget/1/Example.txt", "from git\n")
	worktree, err := repository.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	if err := worktree.AddWithOptions(&git.AddOptions{All: true}); err != nil {
		t.Fatal(err)
	}
	hash, err := worktree.Commit("initial", &git.CommitOptions{Author: &object.Signature{Name: "Archbase Test", Email: "test@archbase.local", When: time.Now().UTC()}})
	if err != nil {
		t.Fatal(err)
	}
	main := plumbing.NewHashReference(plumbing.NewBranchReferenceName("main"), hash)
	if err := repository.Storer.SetReference(main); err != nil {
		t.Fatal(err)
	}
	if err := worktree.Checkout(&git.CheckoutOptions{Branch: main.Name()}); err != nil {
		t.Fatal(err)
	}
	return root
}

func fileRegistryURL(value string) string {
	slashed := filepath.ToSlash(value)
	if !strings.HasPrefix(slashed, "/") {
		slashed = "/" + slashed
	}
	return (&url.URL{Scheme: "file", Path: slashed}).String()
}
