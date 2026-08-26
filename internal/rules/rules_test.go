package rules

import (
	"context"
	"errors"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/EnzoCaetano015/Archbase/internal/contracts"
	"github.com/EnzoCaetano015/Archbase/internal/registry"
	registrydata "github.com/EnzoCaetano015/Archbase/registry"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestRuleIDIsStrictAndCaseSensitive(t *testing.T) {
	if id, err := ParseRuleID("architecture/next-modular@1"); err != nil || id.String() != "architecture/next-modular@1" {
		t.Fatalf("valid ID failed: %q, %v", id, err)
	}
	for _, value := range []string{"Architecture/next@1", "architecture/next", "architecture//next@1", "architecture/next@"} {
		if _, err := ParseRuleID(value); err == nil {
			t.Fatalf("expected invalid ID %q", value)
		}
	}
}

func TestEmbeddedCatalogResolvesOfficialRulesAndPatterns(t *testing.T) {
	ruleSource, err := NewEmbeddedSource()
	if err != nil {
		t.Fatal(err)
	}
	patternSource, err := registry.NewEmbeddedSource()
	if err != nil {
		t.Fatal(err)
	}
	patternResolver, _ := registry.NewResolver(patternSource)
	resolver, err := NewResolver(patternResolver, ruleSource)
	if err != nil {
		t.Fatal(err)
	}
	listed, err := resolver.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"architecture/dotnet-layered@1", "architecture/next-modular@1"}
	if len(listed.Entries) != len(want) {
		t.Fatalf("unexpected rules: %#v", listed.Entries)
	}
	for index, rawID := range want {
		if listed.Entries[index].ID.String() != rawID {
			t.Fatalf("rules are not deterministic: %#v", listed.Entries)
		}
		resolved, err := resolver.Resolve(context.Background(), rawID)
		if err != nil {
			t.Fatalf("resolve %s: %v", rawID, err)
		}
		if len(resolved.Rule.Definition.Scopes) < 3 || len(resolved.Rule.Content) == 0 || resolved.Rule.Source != "official-embedded" {
			t.Fatalf("unexpected rule %s: %#v", rawID, resolved.Rule)
		}
	}
}

func TestOfficialRulesHaveReadmesWithoutPatternSourceCopies(t *testing.T) {
	readmes := 0
	err := fs.WalkDir(registrydata.FS, "rules", func(filePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		switch strings.ToLower(filepath.Ext(filePath)) {
		case ".cs", ".ts", ".tsx":
			t.Fatalf("rule catalog duplicates pattern source: %s", filePath)
		}
		if entry.Name() == "README.md" {
			readmes++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if readmes != 2 {
		t.Fatalf("expected one README for each official rule, got %d", readmes)
	}
}

func TestCatalogRejectsDuplicateTraversalMismatchAndInvalidRule(t *testing.T) {
	tests := []struct {
		name  string
		index string
		rule  string
		want  string
	}{
		{name: "duplicate", index: ruleIndex("architecture/example@1", "architecture/example@1", "architecture/example/1"), rule: testRule("architecture/example@1", "next/page@1234"), want: "duplicate"},
		{name: "traversal", index: ruleIndex("architecture/example@1", "", "../outside"), rule: testRule("architecture/example@1", "next/page@1234"), want: "escapes"},
		{name: "ID mismatch", index: ruleIndex("architecture/example@1", "", "architecture/example/1"), rule: testRule("architecture/other@1", "next/page@1234"), want: "does not match"},
		{name: "version mismatch", index: ruleIndex("architecture/example@1", "", "architecture/example/1"), rule: strings.Replace(testRule("architecture/example@1", "next/page@1234"), "version: 1.0.0", "version: 2.0.0", 1), want: "does not match"},
		{name: "invalid rule", index: ruleIndex("architecture/example@1", "", "architecture/example/1"), rule: strings.Replace(testRule("architecture/example@1", "next/page@1234"), "version: 1.0.0", "version: invalid", 1), want: "invalid rule"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fsys := fstest.MapFS{"index.yaml": &fstest.MapFile{Data: []byte(test.index)}}
			if !strings.Contains(test.index, "../outside") {
				fsys["architecture/example/1/rule.yaml"] = &fstest.MapFile{Data: []byte(test.rule)}
			}
			if _, err := newCatalogSource("test", fsys); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
		})
	}
}

func TestCatalogRejectsUnsortedIndex(t *testing.T) {
	index := "schemaVersion: 1\nrules:\n" +
		"  - id: architecture/z@1\n    version: 1.0.0\n    path: architecture/z/1\n" +
		"  - id: architecture/a@1\n    version: 1.0.0\n    path: architecture/a/1\n"
	fsys := fstest.MapFS{
		"index.yaml":                 &fstest.MapFile{Data: []byte(index)},
		"architecture/z/1/rule.yaml": &fstest.MapFile{Data: []byte(testRule("architecture/z@1", "next/page@1234"))},
		"architecture/a/1/rule.yaml": &fstest.MapFile{Data: []byte(testRule("architecture/a@1", "next/page@1234"))},
	}
	if _, err := newCatalogSource("test", fsys); err == nil || !strings.Contains(err.Error(), "sorted") {
		t.Fatalf("expected ordering failure, got %v", err)
	}
}

func TestDirectorySourceLookupAndNotFound(t *testing.T) {
	root := t.TempDir()
	writeRuleCatalog(t, root, "architecture/example@1", "next/page@1234")
	source, err := NewDirectorySource(root)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := ParseRuleID("architecture/example@1")
	result, err := source.Lookup(context.Background(), id)
	if err != nil || result.Rule.Definition.ID != id.String() {
		t.Fatalf("unexpected lookup: %#v, %v", result, err)
	}
	missing, _ := ParseRuleID("architecture/missing@1")
	if _, err := source.Lookup(context.Background(), missing); !errors.Is(err, ErrRuleNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestDirectorySourceRejectsSymlinkedRuleDocument(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires privileges on many Windows environments")
	}
	root := t.TempDir()
	writeRuleCatalog(t, root, "architecture/example@1", "next/page@1234")
	rulePath := filepath.Join(root, "rules", "architecture", "example", "1", "rule.yaml")
	realPath := filepath.Join(root, "real-rule.yaml")
	if err := os.Rename(rulePath, realPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realPath, rulePath); err != nil {
		t.Fatal(err)
	}
	if _, err := NewDirectorySource(root); err == nil || !strings.Contains(err.Error(), "symbolic links") {
		t.Fatalf("expected symlink rejection, got %v", err)
	}
}

type stubSource struct {
	name   string
	lookup func(context.Context, RuleID) (LookupResult, error)
	list   func(context.Context) (SourceListResult, error)
}

func (s stubSource) Name() string { return s.name }
func (s stubSource) Lookup(ctx context.Context, id RuleID) (LookupResult, error) {
	return s.lookup(ctx, id)
}
func (s stubSource) List(ctx context.Context) (SourceListResult, error) {
	if s.list == nil {
		return SourceListResult{}, nil
	}
	return s.list(ctx)
}

func TestResolverPrecedenceCorruptionReferencesAndWarnings(t *testing.T) {
	patternSource, _ := registry.NewEmbeddedSource()
	patternResolver, _ := registry.NewResolver(patternSource)
	id, _ := ParseRuleID("architecture/example@1")
	makeRule := func(pattern, marker string) Rule {
		definition := mustLoadTestRule(t, testRule(id.String(), pattern))
		definition.Description = marker
		return Rule{Entry: Entry{ID: id, Version: "1.0.0", Source: marker}, Definition: definition, Source: marker}
	}
	t.Run("first source wins and warning propagates", func(t *testing.T) {
		warning := errors.New("stale remote")
		first := stubSource{name: "first", lookup: func(context.Context, RuleID) (LookupResult, error) {
			return LookupResult{Rule: makeRule("next/page@1234", "first"), Stale: true, Warning: warning}, nil
		}}
		second := stubSource{name: "second", lookup: func(context.Context, RuleID) (LookupResult, error) {
			return LookupResult{Rule: makeRule("next/page@1234", "second")}, nil
		}}
		resolver, _ := NewResolver(patternResolver, first, second)
		resolved, err := resolver.Resolve(context.Background(), id.String())
		if err != nil || resolved.Rule.Source != "first" || !resolved.Stale || len(resolved.Warnings) != 1 {
			t.Fatalf("unexpected resolution: %#v, %v", resolved, err)
		}
	})
	t.Run("corruption stops fallback", func(t *testing.T) {
		corrupt := stubSource{name: "corrupt", lookup: func(context.Context, RuleID) (LookupResult, error) {
			return LookupResult{}, errors.New("corrupt index")
		}}
		fallback := stubSource{name: "fallback", lookup: func(context.Context, RuleID) (LookupResult, error) {
			return LookupResult{Rule: makeRule("next/page@1234", "fallback")}, nil
		}}
		resolver, _ := NewResolver(patternResolver, corrupt, fallback)
		if _, err := resolver.Resolve(context.Background(), id.String()); err == nil || !strings.Contains(err.Error(), "corrupt index") {
			t.Fatalf("expected hard error, got %v", err)
		}
	})
	t.Run("missing referenced pattern fails", func(t *testing.T) {
		source := stubSource{name: "source", lookup: func(context.Context, RuleID) (LookupResult, error) {
			return LookupResult{Rule: makeRule("missing/pattern@1", "source")}, nil
		}}
		resolver, _ := NewResolver(patternResolver, source)
		if _, err := resolver.Resolve(context.Background(), id.String()); err == nil || !strings.Contains(err.Error(), "unavailable pattern") {
			t.Fatalf("expected missing-pattern error, got %v", err)
		}
	})
}

func TestResolverListDeduplicatesByPrecedence(t *testing.T) {
	patternSource, _ := registry.NewEmbeddedSource()
	patternResolver, _ := registry.NewResolver(patternSource)
	firstID, _ := ParseRuleID("architecture/a@1")
	secondID, _ := ParseRuleID("architecture/b@1")
	warning := errors.New("stale")
	first := stubSource{name: "first", lookup: nil, list: func(context.Context) (SourceListResult, error) {
		return SourceListResult{Entries: []Entry{{ID: secondID, Source: "first"}, {ID: firstID, Source: "first"}}, Stale: true, Warning: warning}, nil
	}}
	second := stubSource{name: "second", lookup: nil, list: func(context.Context) (SourceListResult, error) {
		return SourceListResult{Entries: []Entry{{ID: firstID, Source: "second"}}}, nil
	}}
	resolver, _ := NewResolver(patternResolver, first, second)
	result, err := resolver.List(context.Background())
	if err != nil || len(result.Entries) != 2 || result.Entries[0].Source != "first" || !result.Stale || len(result.Warnings) != 1 {
		t.Fatalf("unexpected list: %#v, %v", result, err)
	}
}

func TestGitSourceUsesSharedCheckoutForRulesAndPatterns(t *testing.T) {
	remote := createCombinedGitRegistry(t)
	config := registry.GitSourceConfig{URL: fileURL(remote), Ref: "main", CacheRoot: t.TempDir(), TTL: registry.DefaultGitCacheTTL}
	patternSource, err := registry.NewGitSource(config)
	if err != nil {
		t.Fatal(err)
	}
	ruleSource, err := NewGitSource(config)
	if err != nil {
		t.Fatal(err)
	}
	patternResolver, _ := registry.NewResolver(patternSource)
	resolver, _ := NewResolver(patternResolver, ruleSource)
	resolved, err := resolver.Resolve(context.Background(), "architecture/test@1")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Rule.Source[:4] != "git:" || resolved.Rule.Definition.Scopes[0].Pattern != "test/item@1" {
		t.Fatalf("unexpected Git rule: %#v", resolved.Rule)
	}
}

type stubCheckoutProvider struct {
	name     string
	snapshot registry.CheckoutSnapshot
	err      error
}

func (s stubCheckoutProvider) Name() string { return s.name }
func (s stubCheckoutProvider) Snapshot(context.Context) (registry.CheckoutSnapshot, error) {
	return s.snapshot, s.err
}

func TestGitRuleSourceHandlesMissingCatalogAndStaleValidation(t *testing.T) {
	id, _ := ParseRuleID("architecture/example@1")
	t.Run("registry without rules is empty", func(t *testing.T) {
		source := newGitSource(stubCheckoutProvider{name: "git:test", snapshot: registry.CheckoutSnapshot{FS: fstest.MapFS{"index.yaml": &fstest.MapFile{Data: []byte("schemaVersion: 1\npatterns: []\n")}}}})
		listed, err := source.List(context.Background())
		if err != nil || len(listed.Entries) != 0 {
			t.Fatalf("unexpected list: %#v, %v", listed, err)
		}
		if _, err := source.Lookup(context.Background(), id); !errors.Is(err, ErrRuleNotFound) {
			t.Fatalf("expected not found, got %v", err)
		}
	})
	t.Run("validated stale catalog carries warning", func(t *testing.T) {
		warning := errors.New("remote offline")
		fsys := fstest.MapFS{
			"rules/index.yaml":                       &fstest.MapFile{Data: []byte(ruleIndex(id.String(), "", "architecture/example/1"))},
			"rules/architecture/example/1/rule.yaml": &fstest.MapFile{Data: []byte(testRule(id.String(), "next/page@1234"))},
		}
		source := newGitSource(stubCheckoutProvider{name: "git:test", snapshot: registry.CheckoutSnapshot{FS: fsys, Stale: true, Warning: warning}})
		result, err := source.Lookup(context.Background(), id)
		if err != nil || !result.Stale || !errors.Is(result.Warning, warning) {
			t.Fatalf("unexpected stale result: %#v, %v", result, err)
		}
	})
	t.Run("corrupt stale catalog fails", func(t *testing.T) {
		fsys := fstest.MapFS{
			"rules/index.yaml": &fstest.MapFile{Data: []byte("invalid: true\n")},
		}
		source := newGitSource(stubCheckoutProvider{name: "git:test", snapshot: registry.CheckoutSnapshot{FS: fsys, Stale: true, Warning: errors.New("offline")}})
		if _, err := source.Lookup(context.Background(), id); err == nil || !strings.Contains(err.Error(), "stale rule registry cache") {
			t.Fatalf("expected stale validation failure, got %v", err)
		}
	})
}

func ruleIndex(id, duplicate, rulePath string) string {
	result := "schemaVersion: 1\nrules:\n  - id: " + id + "\n    version: 1.0.0\n    path: " + rulePath + "\n"
	if duplicate != "" {
		result += "  - id: " + duplicate + "\n    version: 1.0.0\n    path: architecture/example/1\n"
	}
	return result
}

func testRule(id, pattern string) string {
	return "schemaVersion: 1\nid: " + id + "\nname: Test rule\nversion: 1.0.0\nscopes:\n  - path: src/**\n    pattern: " + pattern + "\nrules:\n  - Keep layers separate.\n"
}

func mustLoadTestRule(t *testing.T, content string) contracts.Rule {
	t.Helper()
	file := filepath.Join(t.TempDir(), "rule.yaml")
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := contracts.LoadRule(file)
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func writeRuleCatalog(t *testing.T, root, id, pattern string) {
	t.Helper()
	writeTestFile(t, filepath.Join(root, "rules", "index.yaml"), ruleIndex(id, "", "architecture/example/1"))
	writeTestFile(t, filepath.Join(root, "rules", "architecture", "example", "1", "rule.yaml"), testRule(id, pattern))
}

func createCombinedGitRegistry(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "remote")
	repository, err := git.PlainInit(root, false)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(root, "index.yaml"), "schemaVersion: 1\npatterns:\n  - id: test/item@1\n    version: 1.0.0\n    path: test/item/1\n")
	manifest := `schemaVersion: 1
id: test/item@1
name: Test item
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
`
	writeTestFile(t, filepath.Join(root, "test", "item", "1", "manifest.yaml"), manifest)
	writeTestFile(t, filepath.Join(root, "test", "item", "1", "Example.txt"), "example\n")
	writeRuleCatalog(t, root, "architecture/test@1", "test/item@1")
	worktree, err := repository.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	if err := worktree.AddWithOptions(&git.AddOptions{All: true}); err != nil {
		t.Fatal(err)
	}
	hash, err := worktree.Commit("initial registry", &git.CommitOptions{Author: &object.Signature{Name: "Archbase Test", Email: "test@archbase.local", When: time.Now().UTC()}})
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

func writeTestFile(t *testing.T, file, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func fileURL(value string) string {
	slashed := filepath.ToSlash(value)
	if !strings.HasPrefix(slashed, "/") {
		slashed = "/" + slashed
	}
	return (&url.URL{Scheme: "file", Path: slashed}).String()
}
