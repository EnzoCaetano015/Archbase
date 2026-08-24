package registry

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestGitSourceCloneCacheHitAndRefresh(t *testing.T) {
	remote, repository, firstCommit := createGitRegistry(t, "first")
	if _, err := repository.CreateTag("v1", firstCommit, nil); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	source := newTestGitSource(t, remote, "main", t.TempDir())
	source.now = func() time.Time { return now }
	id, _ := ParsePatternID("test/item@1")

	first, err := source.Lookup(context.Background(), id)
	if err != nil || bundleContent(first) != "first" || first.Stale {
		t.Fatalf("unexpected initial clone: %#v, %v", first, err)
	}
	updateGitRegistry(t, remote, "second")

	cached, err := source.Lookup(context.Background(), id)
	if err != nil || bundleContent(cached) != "first" {
		t.Fatalf("TTL cache was not used: %#v, %v", cached, err)
	}
	now = now.Add(DefaultGitCacheTTL + time.Second)
	refreshed, err := source.Lookup(context.Background(), id)
	if err != nil || bundleContent(refreshed) != "second" || refreshed.Stale {
		t.Fatalf("stale cache was not refreshed: %#v, %v", refreshed, err)
	}
	if _, err := os.Stat(source.lockPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cache lock was left behind: %v", err)
	}
}

func TestGitSourceSelectsTag(t *testing.T) {
	remote, repository, firstCommit := createGitRegistry(t, "tagged")
	if _, err := repository.CreateTag("v1", firstCommit, nil); err != nil {
		t.Fatal(err)
	}
	updateGitRegistry(t, remote, "main")
	source := newTestGitSource(t, remote, "v1", t.TempDir())
	id, _ := ParsePatternID("test/item@1")
	result, err := source.Lookup(context.Background(), id)
	if err != nil || bundleContent(result) != "tagged" {
		t.Fatalf("tag was not selected: %#v, %v", result, err)
	}
}

func TestGitSourceListsCachedRegistry(t *testing.T) {
	remote, _, _ := createGitRegistry(t, "listed")
	source := newTestGitSource(t, remote, "main", t.TempDir())
	result, err := source.List(context.Background())
	if err != nil || len(result.Entries) != 1 || result.Entries[0].ID.String() != "test/item@1" {
		t.Fatalf("unexpected list result: %#v, %v", result, err)
	}
}

func TestGitSourceUsesConfiguredSubdirectory(t *testing.T) {
	remote, repository, _ := createGitRegistry(t, "nested")
	catalogRoot := filepath.Join(remote, "catalog")
	if err := os.MkdirAll(catalogRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(remote, "index.yaml"), filepath.Join(catalogRoot, "index.yaml")); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(remote, "test"), filepath.Join(catalogRoot, "test")); err != nil {
		t.Fatal(err)
	}
	commitRegistry(t, repository, "move registry to subdirectory")
	source, err := NewGitSource(GitSourceConfig{
		URL: fileURL(remote), Ref: "main", Subdirectory: "catalog",
		CacheRoot: t.TempDir(), TTL: DefaultGitCacheTTL,
	})
	if err != nil {
		t.Fatal(err)
	}
	id, _ := ParsePatternID("test/item@1")
	result, err := source.Lookup(context.Background(), id)
	if err != nil || bundleContent(result) != "nested" {
		t.Fatalf("configured subdirectory was not used: %#v, %v", result, err)
	}
}

func TestGitSourceUsesValidatedStaleCacheWhenRemoteFails(t *testing.T) {
	remote, _, _ := createGitRegistry(t, "cached")
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	source := newTestGitSource(t, remote, "main", t.TempDir())
	source.now = func() time.Time { return now }
	id, _ := ParsePatternID("test/item@1")
	if _, err := source.Lookup(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(remote, remote+"-offline"); err != nil {
		t.Fatal(err)
	}
	now = now.Add(DefaultGitCacheTTL + time.Second)
	result, err := source.Lookup(context.Background(), id)
	if err != nil || !result.Stale || result.Warning == nil || bundleContent(result) != "cached" {
		t.Fatalf("expected validated stale fallback, got %#v, %v", result, err)
	}
}

func TestGitSourceRejectsMissingOrCorruptedCache(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "missing.git")
		source := newTestGitSource(t, missing, "main", t.TempDir())
		id, _ := ParsePatternID("test/item@1")
		if _, err := source.Lookup(context.Background(), id); err == nil {
			t.Fatal("expected failure without remote or cache")
		}
	})
	t.Run("corrupted stale", func(t *testing.T) {
		remote, _, _ := createGitRegistry(t, "cached")
		now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
		source := newTestGitSource(t, remote, "main", t.TempDir())
		source.now = func() time.Time { return now }
		id, _ := ParsePatternID("test/item@1")
		if _, err := source.Lookup(context.Background(), id); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(source.config.registryPath(), "index.yaml"), "not: a registry\n")
		if err := os.Rename(remote, remote+"-offline"); err != nil {
			t.Fatal(err)
		}
		now = now.Add(DefaultGitCacheTTL + time.Second)
		if _, err := source.Lookup(context.Background(), id); err == nil || !strings.Contains(err.Error(), "cache") {
			t.Fatalf("expected corrupted cache failure, got %v", err)
		}
	})
}

func TestGitSourceSerializesConcurrentClone(t *testing.T) {
	remote, _, _ := createGitRegistry(t, "concurrent")
	source := newTestGitSource(t, remote, "main", t.TempDir())
	id, _ := ParsePatternID("test/item@1")
	start := make(chan struct{})
	errorsFound := make(chan error, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			result, err := source.Lookup(context.Background(), id)
			if err == nil && bundleContent(result) != "concurrent" {
				err = errors.New("unexpected bundle content")
			}
			errorsFound <- err
		}()
	}
	close(start)
	wait.Wait()
	close(errorsFound)
	for err := range errorsFound {
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(source.lockPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cache lock was left behind: %v", err)
	}
}

func TestGitSourceLockHonorsContextCancellation(t *testing.T) {
	remote, _, _ := createGitRegistry(t, "locked")
	source := newTestGitSource(t, remote, "main", t.TempDir())
	if err := os.MkdirAll(source.config.CacheRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, source.lockPath(), "owned elsewhere")
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Millisecond)
	defer cancel()
	id, _ := ParsePatternID("test/item@1")
	_, err := source.Lookup(ctx, id)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline, got %v", err)
	}
}

func TestGitSourceRecoversExpiredLock(t *testing.T) {
	remote, _, _ := createGitRegistry(t, "unlocked")
	source := newTestGitSource(t, remote, "main", t.TempDir())
	if err := os.MkdirAll(source.config.CacheRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, source.lockPath(), "abandoned")
	old := time.Now().Add(-cacheLockMaxAge - time.Minute)
	if err := os.Chtimes(source.lockPath(), old, old); err != nil {
		t.Fatal(err)
	}
	release, err := source.acquireLock(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	release()
	if _, err := os.Stat(source.lockPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("recovered lock was left behind: %v", err)
	}
}

func TestGitSourceConfigSecurityAndCacheKey(t *testing.T) {
	cacheRoot, _ := filepath.Abs(t.TempDir())
	valid := GitSourceConfig{URL: "https://github.com/example/registry.git", Ref: "main", CacheRoot: cacheRoot}
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, rawURL := range []string{
		"http://example.com/registry.git",
		"ssh://example.com/registry.git",
		"https://token@example.com/registry.git",
		"file:relative/path",
	} {
		config := valid
		config.URL = rawURL
		if err := config.Validate(); err == nil {
			t.Fatalf("expected URL %q to be rejected", rawURL)
		}
	}
	if strings.Contains(valid.checkoutPath(), "github") || strings.Contains(valid.checkoutPath(), "registry.git") {
		t.Fatalf("cache path exposes URL: %s", valid.checkoutPath())
	}
	config := valid
	config.Subdirectory = "../outside"
	if err := config.Validate(); err == nil {
		t.Fatal("expected invalid subdirectory")
	}
}

func newTestGitSource(t *testing.T, remote, ref, cacheRoot string) *GitSource {
	t.Helper()
	source, err := NewGitSource(GitSourceConfig{
		URL:       fileURL(remote),
		Ref:       ref,
		CacheRoot: cacheRoot,
		TTL:       DefaultGitCacheTTL,
	})
	if err != nil {
		t.Fatal(err)
	}
	return source
}

func createGitRegistry(t *testing.T, content string) (string, *git.Repository, plumbing.Hash) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "remote")
	repository, err := git.PlainInit(root, false)
	if err != nil {
		t.Fatal(err)
	}
	worktree, err := repository.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	writeRegistryTree(t, root, content)
	hash := commitRegistry(t, repository, "initial registry")
	mainReference := plumbing.NewHashReference(plumbing.NewBranchReferenceName("main"), hash)
	if err := repository.Storer.SetReference(mainReference); err != nil {
		t.Fatal(err)
	}
	if err := worktree.Checkout(&git.CheckoutOptions{Branch: mainReference.Name()}); err != nil {
		t.Fatal(err)
	}
	return root, repository, hash
}

func updateGitRegistry(t *testing.T, root, content string) plumbing.Hash {
	t.Helper()
	repository, err := git.PlainOpen(root)
	if err != nil {
		t.Fatal(err)
	}
	writeRegistryTree(t, root, content)
	return commitRegistry(t, repository, "update registry")
}

func writeRegistryTree(t *testing.T, root, content string) {
	t.Helper()
	index := "schemaVersion: 1\npatterns:\n  - id: test/item@1\n    version: 1.0.0\n    path: test/item/1\n"
	writeFile(t, filepath.Join(root, "index.yaml"), index)
	writePattern(t, filepath.Join(root, "test", "item", "1"), "test/item@1", content)
}

func commitRegistry(t *testing.T, repository *git.Repository, message string) plumbing.Hash {
	t.Helper()
	worktree, err := repository.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	if err := worktree.AddWithOptions(&git.AddOptions{All: true}); err != nil {
		t.Fatal(err)
	}
	hash, err := worktree.Commit(message, &git.CommitOptions{Author: &object.Signature{
		Name: "Archbase Test", Email: "test@archbase.local", When: time.Now().UTC(),
	}})
	if err != nil {
		t.Fatal(err)
	}
	return hash
}

func fileURL(path string) string {
	slashed := filepath.ToSlash(path)
	if !strings.HasPrefix(slashed, "/") {
		slashed = "/" + slashed
	}
	return (&url.URL{Scheme: "file", Path: slashed}).String()
}

func bundleContent(result LookupResult) string {
	if len(result.Pattern.Bundle.Files) == 0 {
		return ""
	}
	return string(result.Pattern.Bundle.Files[0].Content)
}
