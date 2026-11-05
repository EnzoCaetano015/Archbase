package diagnostics

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func linkOrSkip(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
}

func TestRootAndDeclaredPathLinksAreRejected(t *testing.T) {
	outside := tempDir(t)
	put(t, outside, "secret.txt", []byte("OUTSIDE_SENTINEL"))
	parent := tempDir(t)
	linked := filepath.Join(parent, "linked")
	linkOrSkip(t, outside, linked)
	if r, err := OpenRoot(context.Background(), linked, DefaultLimits); err == nil {
		r.Close()
		t.Fatal("linked project root accepted")
	}
	if r, err := OpenRoot(context.Background(), filepath.Join(linked, "child"), DefaultLimits); err == nil {
		r.Close()
		t.Fatal("linked ancestor accepted")
	}
	root := tempDir(t)
	linkOrSkip(t, outside, filepath.Join(root, "outside"))
	r, err := OpenRoot(context.Background(), root, DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	for _, name := range []string{"outside/secret.txt", "../secret.txt", "/secret.txt", "C:/secret.txt", "outside\\secret.txt", "file:stream"} {
		if data, err := r.ReadFile(name); err == nil || len(data) > 0 {
			t.Fatalf("unsafe read %q: %q %v", name, data, err)
		}
	}
}

func TestPathSwapCannotReadOutsideRoot(t *testing.T) {
	root := tempDir(t)
	outside := tempDir(t)
	put(t, outside, "secret.txt", []byte("OUTSIDE_SENTINEL"))
	put(t, root, "file.md", []byte("original"))
	probe := filepath.Join(root, "probe")
	linkOrSkip(t, filepath.Join(outside, "secret.txt"), probe)
	if err := os.Remove(probe); err != nil {
		t.Fatal(err)
	}
	raw, err := OpenRoot(context.Background(), root, DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	r := raw.(*rootReader)
	r.beforeOpen = func(name string) {
		if name != "file.md" {
			return
		}
		if err := os.Remove(filepath.Join(root, "file.md")); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(root, "file.md")); err != nil {
			t.Fatal(err)
		}
	}
	if data, err := r.ReadFile("file.md"); err == nil || len(data) > 0 {
		t.Fatalf("path-swap leaked outside bytes: %q %v", data, err)
	}
}

func TestMutationAndBudgetCannotBecomeSuccess(t *testing.T) {
	root := tempDir(t)
	put(t, root, "file.md", []byte("original"))
	raw, err := OpenRoot(context.Background(), root, Limits{Files: 20, FileBytes: 8, TotalBytes: 8})
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	if _, err = raw.ReadFile("file.md"); err != nil {
		t.Fatal(err)
	}
	if _, err = raw.ReadFile("file.md"); !errors.Is(err, ErrBudget) {
		t.Fatalf("total budget not enforced: %v", err)
	}
	put(t, root, "file.md", []byte("changed contents"))
	if err = raw.Verify(); !errors.Is(err, ErrChanged) {
		t.Fatalf("mutation undetected: %v", err)
	}
}

func TestDirectoryCacheCannotHideChangesOrTruncate(t *testing.T) {
	root := tempDir(t)
	put(t, root, "one.md", []byte("one"))
	put(t, root, "two.md", []byte("two"))
	r, err := OpenRoot(context.Background(), root, Limits{Files: 1, FileBytes: 10, TotalBytes: 20})
	if err != nil {
		t.Fatal(err)
	}
	if entries, err := r.ReadDir("."); !errors.Is(err, ErrBudget) || entries != nil {
		t.Fatal("oversized listing was silently truncated")
	}
	r.Close()
	r, err = OpenRoot(context.Background(), root, DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	entries, err := r.ReadDir(".")
	if err != nil || len(entries) != 2 {
		t.Fatal(err)
	}
	put(t, root, "new.md", []byte("new"))
	if _, err = r.ReadDir("."); !errors.Is(err, ErrChanged) {
		t.Fatalf("changed directory cache accepted: %v", err)
	}
}

func TestCancellationDuringReadIsNotReportedAsSuccess(t *testing.T) {
	root := tempDir(t)
	put(t, root, "file.md", []byte("body"))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r, err := OpenRoot(ctx, root, DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	r.(*rootReader).beforeOpen = func(string) { cancel() }
	if data, err := r.ReadFile("file.md"); !errors.Is(err, context.Canceled) || len(data) > 0 {
		t.Fatalf("%q %v", data, err)
	}
	if err = r.Verify(); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestDeclaredSecretPathIsExcludedBeforeOpen(t *testing.T) {
	root, ws := fixture(t)
	base := filepath.Join(root, ".archbase", filepath.FromSlash(ws.Scopes[0].Pattern.Root))
	manifestPath := filepath.Join(base, "manifest.yaml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	// Substitute a source name in an otherwise valid manifest, preserving its
	// typed destination. The exclusion must happen before any content read.
	text := string(data)
	text = replaceFirst(text, "Example/Example.tsx.md", ".env.tsx.md")
	put(t, base, "manifest.yaml", []byte(text))
	put(t, base, ".env.tsx.md", []byte("DO_NOT_READ"))
	openedSecret := false
	r := Run(context.Background(), root, "dev", Options{Open: func(ctx context.Context, p string, l Limits) (Reader, error) {
		r, e := OpenRoot(ctx, p, l)
		if e == nil {
			r.(*rootReader).beforeOpen = func(name string) {
				if filepath.Base(name) == ".env.tsx.md" {
					openedSecret = true
				}
			}
		}
		return r, e
	}})
	if openedSecret || check(t, r, "local-bundles").Reason != "excluded-sensitive-path" || r.ExitCode() != 3 {
		t.Fatalf("secret path was not excluded: %+v", r)
	}
}

func replaceFirst(s, old, new string) string {
	for i := 0; i+len(old) <= len(s); i++ {
		if s[i:i+len(old)] == old {
			return s[:i] + new + s[i+len(old):]
		}
	}
	return s
}
