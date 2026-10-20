package workspace

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/EnzoCaetano015/Archbase/internal/contracts"
	"github.com/EnzoCaetano015/Archbase/internal/patterns"
	"github.com/EnzoCaetano015/Archbase/internal/registry"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func testService(t *testing.T, fsys FileSystem) *Service {
	t.Helper()
	source, err := registry.NewEmbeddedSource()
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := registry.NewResolver(source)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(fsys, resolver)
	if err != nil {
		t.Fatal(err)
	}
	return service
}
func newWorkspace(t *testing.T, fsys FileSystem) (*Service, string) {
	t.Helper()
	root := t.TempDir()
	return testService(t, fsys).WithProjectRoot(root), root
}
func loadIndex(t *testing.T, root string) contracts.Workspace {
	t.Helper()
	index, err := contracts.LoadWorkspaceFS(os.DirFS(filepath.Join(root, ".archbase")), "scopes.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return index
}
func saveIndex(t *testing.T, root string, index contracts.Workspace) {
	t.Helper()
	data, err := contracts.EncodeWorkspace(index)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, ".archbase", "scopes.yaml"), data, 0644); err != nil {
		t.Fatal(err)
	}
}
func TestCentralWorkspaceInstallationAndIndependentScopes(t *testing.T) {
	s, root := newWorkspace(t, OSFileSystem{})
	a := filepath.Join(root, "src", "pages")
	b := filepath.Join(root, "src", "admin")
	first, err := s.Add(context.Background(), "next/page@1234", a)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.Add(context.Background(), "next/page@1234", b)
	if err != nil {
		t.Fatal(err)
	}
	expected := filepath.Join(root, ".archbase", "patterns", fmt.Sprintf("%x", sha256.Sum256([]byte("src/pages"))), fmt.Sprintf("%x", sha256.Sum256([]byte("next/page@1234"))))
	if first.PatternDirectory != expected || first.PatternDirectory == second.PatternDirectory {
		t.Fatalf("wrong storage: %#v %#v", first, second)
	}
	index := loadIndex(t, root)
	if len(index.Scopes) != 2 || index.Scopes[1].Origin.Registry != "official-embedded" {
		t.Fatalf("%#v", index)
	}
	custom := []byte("export const customized = true;\n")
	f := filepath.Join(first.PatternDirectory, "Example", "Example.tsx.md")
	if err = os.WriteFile(f, patterns.EncodeDocument("Example.tsx", custom), 0644); err != nil {
		t.Fatal(err)
	}
	r, err := s.Resolve(context.Background(), filepath.Join(a, "future.tsx"))
	if err != nil || string(r.Pattern.Bundle.Files[0].Content) != string(custom) {
		t.Fatalf("%#v %v", r, err)
	}
	other, err := s.Resolve(context.Background(), b)
	if err != nil || string(other.Pattern.Bundle.Files[0].Content) == string(custom) {
		t.Fatalf("coupled copies: %v", err)
	}
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			t.Fatal(e)
		}
		if d.Name() == ".archbase" && p != filepath.Join(root, ".archbase") {
			t.Fatalf("nested storage %s", p)
		}
		return nil
	})
	if _, err = os.Stat(filepath.Join(first.PatternDirectory, "README.md")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("README installed")
	}
	if _, err = s.Add(context.Background(), "next/page@1234", a); !errors.Is(err, ErrPatternExists) {
		t.Fatalf("collision: %v", err)
	}
}
func TestCreateMinimalDerivedAndPreserveInactive(t *testing.T) {
	s, root := newWorkspace(t, OSFileSystem{})
	a, err := s.Create(context.Background(), "minimal", root, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(a.PatternDirectory, "Example.txt.md")); err != nil {
		t.Fatal(err)
	}
	b, err := s.Create(context.Background(), "pages-standard", root, "next/page@1234")
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.Resolve(context.Background(), root)
	if err != nil || r.Pattern.Bundle.Manifest.ID != "local/pages-standard@1" || r.Scope.Origin.ID != "next/page@1234" {
		t.Fatalf("%#v %v", r, err)
	}
	if _, err = os.Stat(a.PatternDirectory); err != nil {
		t.Fatal(err)
	}
	if b.PatternID != "local/pages-standard@1" {
		t.Fatal(b)
	}
	if _, err = s.Create(context.Background(), "Pages Standard", root, ""); err == nil {
		t.Fatal("invalid name accepted")
	}
}
func TestNearestScopeFilesMissingAndDirectoryBoundaries(t *testing.T) {
	s, root := newWorkspace(t, OSFileSystem{})
	if _, err := s.Create(context.Background(), "parent", root, ""); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(root, "src", "pages")
	if _, err := s.Create(context.Background(), "child", child, ""); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(child, "existing.ts")
	if err := os.WriteFile(file, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{child, file, filepath.Join(child, "future", "file.ts")} {
		r, err := s.WithProjectRoot("").Resolve(context.Background(), target)
		if err != nil || r.Pattern.Entry.ID != "local/child@1" || r.ScopeDirectory != child {
			t.Fatalf("%s %#v %v", target, r, err)
		}
	}
	r, err := s.Resolve(context.Background(), filepath.Join(root, "src", "pages-old", "file.ts"))
	if err != nil || r.Pattern.Entry.ID != "local/parent@1" {
		t.Fatalf("%#v %v", r, err)
	}
}
func TestCurrentDirectoryIsWriteRoot(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	s := testService(t, OSFileSystem{})
	if _, err := s.Create(context.Background(), "example", "src/pages", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".archbase", "scopes.yaml")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(context.Background(), "outside", t.TempDir(), ""); err == nil {
		t.Fatal("outside scope accepted")
	}
}
func TestRegistryScopeAndInvalidIndex(t *testing.T) {
	s, root := newWorkspace(t, OSFileSystem{})
	if err := os.Mkdir(filepath.Join(root, ".archbase"), 0755); err != nil {
		t.Fatal(err)
	}
	scope := contracts.Scope{SchemaVersion: 1, Scope: contracts.ScopeSelector{Path: "."}, Pattern: contracts.ScopePattern{ID: "next/hook@9214", Source: "registry"}, Behavior: contracts.ScopeBehavior{NearestScopeWins: true}}
	saveIndex(t, root, contracts.Workspace{SchemaVersion: 1, Scopes: []contracts.Scope{scope}})
	r, err := s.Resolve(context.Background(), root)
	if err != nil || r.Pattern.Entry.ID != "next/hook@9214" {
		t.Fatalf("%#v %v", r, err)
	}
	if err = os.WriteFile(filepath.Join(root, ".archbase", "scopes.yaml"), []byte("invalid: true\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Resolve(context.Background(), root); !errors.Is(err, ErrInvalidScope) {
		t.Fatal(err)
	}
}
func TestRevalidateClosestBundleAndListing(t *testing.T) {
	for _, corruption := range []string{"missing", "mismatch", "escape", "markdown"} {
		t.Run(corruption, func(t *testing.T) {
			s, root := newWorkspace(t, OSFileSystem{})
			if _, err := s.Create(context.Background(), "parent", root, ""); err != nil {
				t.Fatal(err)
			}
			a, err := s.Create(context.Background(), "child", filepath.Join(root, "nested"), "")
			if err != nil {
				t.Fatal(err)
			}
			switch corruption {
			case "missing":
				if err = os.Remove(filepath.Join(a.PatternDirectory, "Example.txt.md")); err != nil {
					t.Fatal(err)
				}
			case "markdown":
				if err = os.WriteFile(filepath.Join(a.PatternDirectory, "Example.txt.md"), []byte("unclosed"), 0644); err != nil {
					t.Fatal(err)
				}
			case "mismatch":
				idx := loadIndex(t, root)
				idx.Scopes[1].Pattern.ID = "local/other@1"
				saveIndex(t, root, idx)
			case "escape":
				idx := loadIndex(t, root)
				file := filepath.Join(root, ".archbase", "scopes.yaml")
				data, err := os.ReadFile(file)
				if err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(file, []byte(strings.ReplaceAll(string(data), idx.Scopes[1].Pattern.Root, "../outside")), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if _, err = s.Resolve(context.Background(), filepath.Join(root, "nested", "future")); err == nil {
				t.Fatal("corrupt child hidden")
			}
			if _, err = s.List(context.Background(), root); err == nil {
				t.Fatal("corrupt list accepted")
			}
		})
	}
}

type scopeWriteFailureFS struct{ OSFileSystem }

func (scopeWriteFailureFS) WriteFileAtomic(p string, b []byte, overwrite bool) error {
	if filepath.Base(p) == "scopes.yaml" && overwrite {
		return errors.New("simulated scope write failure")
	}
	return archfsWrite(p, b, overwrite)
}
func archfsWrite(p string, b []byte, overwrite bool) error {
	return (OSFileSystem{}).WriteFileAtomic(p, b, overwrite)
}
func TestFailedScopeUpdateRollsBackNewPattern(t *testing.T) {
	s, root := newWorkspace(t, OSFileSystem{})
	if _, err := s.Create(context.Background(), "initial", root, ""); err != nil {
		t.Fatal(err)
	}
	faulty := testService(t, scopeWriteFailureFS{}).WithProjectRoot(root)
	if _, err := faulty.Add(context.Background(), "next/hook@9214", root); err == nil {
		t.Fatal("failure not injected")
	}
	key := fmt.Sprintf("patterns/%x/%x", sha256.Sum256([]byte(".")), sha256.Sum256([]byte("next/hook@9214")))
	if _, err := os.Stat(filepath.Join(root, ".archbase", filepath.FromSlash(key))); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("partial bundle %v", err)
	}
	r, err := s.Resolve(context.Background(), root)
	if err != nil || r.Pattern.Entry.ID != "local/initial@1" {
		t.Fatalf("%#v %v", r, err)
	}
}
func TestConcurrentWritesAndCancellation(t *testing.T) {
	s, root := newWorkspace(t, OSFileSystem{})
	var wg sync.WaitGroup
	errs := make(chan error, 6)
	for i := range 6 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.Create(context.Background(), fmt.Sprintf("pattern-%d", i), filepath.Join(root, fmt.Sprintf("scope-%d", i)), "")
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(loadIndex(t, root).Scopes) != 6 {
		t.Fatal("lost scope")
	}
	unlock, err := s.lock(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if _, err = s.Create(ctx, "cancelled", root, ""); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	unlock()
	if _, err = os.Stat(filepath.Join(root, ".archbase.lock")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("lock leaked")
	}
}
func TestRejectSymlinkComponents(t *testing.T) {
	s, root := newWorkspace(t, OSFileSystem{})
	outside := t.TempDir()
	link := filepath.Join(root, "link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := s.Create(context.Background(), "bad", filepath.Join(link, "nested"), ""); err == nil {
		t.Fatal("symlink accepted")
	}
}

func TestRejectSymlinkedIndexAndPatternRoot(t *testing.T) {
	for _, kind := range []string{"index", "bundle"} {
		t.Run(kind, func(t *testing.T) {
			s, root := newWorkspace(t, OSFileSystem{})
			installed, err := s.Create(context.Background(), "example", root, "")
			if err != nil {
				t.Fatal(err)
			}
			original := filepath.Join(root, ".archbase", "scopes.yaml")
			if kind == "bundle" {
				original = installed.PatternDirectory
			}
			moved := filepath.Join(t.TempDir(), "original")
			if err = os.Rename(original, moved); err != nil {
				t.Fatal(err)
			}
			if err = os.Symlink(moved, original); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			if _, err = s.Resolve(context.Background(), root); err == nil {
				t.Fatal("symlink accepted")
			}
			if _, err = s.List(context.Background(), root); err == nil {
				t.Fatal("symlink accepted by listing")
			}
		})
	}
}

type promotionFailureFS struct{ OSFileSystem }

func (promotionFailureFS) Rename(a, b string) error { return errors.New("simulated promotion failure") }
func TestFirstInstallationFailureLeavesNoWorkspaceOrStaging(t *testing.T) {
	s, root := newWorkspace(t, promotionFailureFS{})
	if _, err := s.Create(context.Background(), "example", root, ""); err == nil {
		t.Fatal("failure not injected")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("partial transaction: %v", entries)
	}
}
func TestAddRawGitRegistryConvertsDocuments(t *testing.T) {
	remote := createWorkspaceGitRegistry(t)
	source, err := registry.NewGitSource(registry.GitSourceConfig{URL: fileRegistryURL(remote), Ref: "main", CacheRoot: t.TempDir(), TTL: registry.DefaultGitCacheTTL})
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := registry.NewResolver(source)
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewService(OSFileSystem{}, resolver)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	s = s.WithProjectRoot(root)
	a, err := s.Add(context.Background(), "custom/widget@1", root)
	if err != nil {
		t.Fatal(err)
	}
	document, err := os.ReadFile(filepath.Join(a.PatternDirectory, "Example.txt.md"))
	if err != nil || !strings.HasPrefix(string(document), "```text") {
		t.Fatalf("%s %v", document, err)
	}
	r, err := s.Resolve(context.Background(), root)
	if err != nil || string(r.Pattern.Bundle.Files[0].Content) != "from git\n" || !strings.HasPrefix(r.Scope.Origin.Registry, "git:") {
		t.Fatalf("%#v %v", r, err)
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
