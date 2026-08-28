package mcp

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/EnzoCaetano015/Archbase/internal/contracts"
	"github.com/EnzoCaetano015/Archbase/internal/patterns"
	"github.com/EnzoCaetano015/Archbase/internal/registry"
	archrules "github.com/EnzoCaetano015/Archbase/internal/rules"
	"github.com/EnzoCaetano015/Archbase/internal/workspace"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func newTestService(t *testing.T, root string) (*Service, *workspace.Service) {
	t.Helper()
	patternSource, err := registry.NewEmbeddedSource()
	if err != nil {
		t.Fatal(err)
	}
	patterns, err := registry.NewResolver(patternSource)
	if err != nil {
		t.Fatal(err)
	}
	ruleSource, err := archrules.NewEmbeddedSource()
	if err != nil {
		t.Fatal(err)
	}
	rules, err := archrules.NewResolver(patterns, ruleSource)
	if err != nil {
		t.Fatal(err)
	}
	workspaceService, err := workspace.NewService(workspace.OSFileSystem{}, patterns)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(root, patterns, rules, workspaceService, OSFileSystem{})
	if err != nil {
		t.Fatal(err)
	}
	return service, workspaceService
}

func TestSearchAndGetEmbeddedPatterns(t *testing.T) {
	service, _ := newTestService(t, t.TempDir())
	all, err := service.SearchPatterns(context.Background(), SearchPatternsInput{})
	if err != nil || len(all.Patterns) != 7 {
		t.Fatalf("unexpected search result: %#v, %v", all, err)
	}
	if !sort.SliceIsSorted(all.Patterns, func(i, j int) bool { return all.Patterns[i].ID < all.Patterns[j].ID }) {
		t.Fatal("patterns are not ordered")
	}
	filtered, err := service.SearchPatterns(context.Background(), SearchPatternsInput{Query: "SERVICE DELEGATION"})
	if err != nil || len(filtered.Patterns) != 1 || filtered.Patterns[0].ID != "dotnet/controller@7743" {
		t.Fatalf("unexpected filtered result: %#v, %v", filtered, err)
	}
	pattern, err := service.GetPattern(context.Background(), PatternIDInput{PatternID: "next/page@1234"})
	if err != nil || pattern.Manifest.Name == "" || pattern.Source != "official-embedded" {
		t.Fatalf("unexpected pattern: %#v, %v", pattern, err)
	}
	if _, err := service.GetPattern(context.Background(), PatternIDInput{PatternID: "next/missing@1"}); !errors.Is(err, registry.ErrPatternNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestResolveFilesRulesAndProjectScopesUseLocalCustomization(t *testing.T) {
	root := t.TempDir()
	service, workspaceService := newTestService(t, root)
	if _, err := workspaceService.Add(context.Background(), "next/page@1234", root); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "src", "pages")
	installed, err := workspaceService.Create(context.Background(), "pages-standard", nested, "next/page@1234")
	if err != nil {
		t.Fatal(err)
	}
	custom := "// locally customized\nexport default function Example() { return null }\n"
	customFile := filepath.Join(installed.PatternDirectory, "Example", "Example.tsx")
	if err := os.WriteFile(customFile, []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(nested, "Admin", "Page.tsx")
	resolved, err := service.ResolvePattern(context.Background(), PathInput{Path: target})
	if err != nil || resolved.Manifest.ID != "local/pages-standard@1" || resolved.ScopePath != "src/pages" {
		t.Fatalf("unexpected resolution: %#v, %v", resolved, err)
	}
	files, err := service.GetPatternFiles(context.Background(), PatternFilesInput{Path: "src/pages/Admin/Page.tsx"})
	if err != nil || files.PatternID != "local/pages-standard@1" || !containsFileContent(files.Files, custom) {
		t.Fatalf("local customization was not returned: %#v, %v", files, err)
	}
	rules, err := service.GetScopeRules(context.Background(), PathInput{Path: "src/pages/Admin/Page.tsx"})
	if err != nil || rules.OriginID != "next/page@1234" || len(rules.Rules) != 1 || rules.Rules[0].ID != "architecture/next-modular@1" {
		t.Fatalf("unexpected scope rules: %#v, %v", rules, err)
	}
	scopes, err := service.ListProjectScopes(context.Background(), EmptyInput{})
	if err != nil || len(scopes.Scopes) != 2 || scopes.Scopes[0].Path != "." || scopes.Scopes[1].Path != "src/pages" {
		t.Fatalf("unexpected project scopes: %#v, %v", scopes, err)
	}
	if _, err := service.GetPatternFiles(context.Background(), PatternFilesInput{}); err == nil {
		t.Fatal("expected missing selector error")
	}
	if _, err := service.GetPatternFiles(context.Background(), PatternFilesInput{PatternID: "next/page@1234", Path: "."}); err == nil {
		t.Fatal("expected mutually exclusive selector error")
	}
}

func TestPatternFilesEncodeBinaryAndPreserveOptionalAbsence(t *testing.T) {
	registryRoot := t.TempDir()
	writeFile(t, filepath.Join(registryRoot, "index.yaml"), "schemaVersion: 1\npatterns:\n  - id: test/binary@1\n    version: 1.0.0\n    path: test/binary/1\n")
	writeFile(t, filepath.Join(registryRoot, "test", "binary", "1", "manifest.yaml"), `schemaVersion: 1
id: test/binary@1
name: Binary pattern
type: pattern-bundle
version: 1.0.0
structure:
  root: "{{Name}}"
  files:
    - source: binary.dat
      destination: "{{Name}}.dat"
      required: true
    - source: optional.txt
      destination: "{{Name}}.txt"
      required: false
allowedChanges: [content]
preserve: [file-responsibility]
`)
	binary := []byte{0xff, 0x00, 0xfe}
	if err := os.WriteFile(filepath.Join(registryRoot, "test", "binary", "1", "binary.dat"), binary, 0o644); err != nil {
		t.Fatal(err)
	}
	source, err := registry.NewDirectorySource(registryRoot)
	if err != nil {
		t.Fatal(err)
	}
	patterns, _ := registry.NewResolver(source)
	ruleSource, _ := archrules.NewEmbeddedSource()
	rules, _ := archrules.NewResolver(patterns, ruleSource)
	workspaceService, _ := workspace.NewService(workspace.OSFileSystem{}, patterns)
	service, err := NewService(t.TempDir(), patterns, rules, workspaceService, OSFileSystem{})
	if err != nil {
		t.Fatal(err)
	}
	output, err := service.GetPatternFiles(context.Background(), PatternFilesInput{PatternID: "test/binary@1"})
	if err != nil || len(output.Files) != 2 {
		t.Fatalf("unexpected files: %#v, %v", output, err)
	}
	if output.Files[0].Encoding != "base64" || output.Files[0].Content != base64.StdEncoding.EncodeToString(binary) {
		t.Fatalf("unexpected binary encoding: %#v", output.Files[0])
	}
	if output.Files[1].Present || output.Files[1].Encoding != "" || output.Files[1].Content != "" {
		t.Fatalf("unexpected optional file: %#v", output.Files[1])
	}
}

func TestProjectPathConfinementAndScopeValidation(t *testing.T) {
	root := t.TempDir()
	service, workspaceService := newTestService(t, root)
	if _, err := workspaceService.Add(context.Background(), "next/page@1234", root); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(filepath.Dir(root), "outside.txt")
	for _, value := range []string{"../outside", outside} {
		if _, err := service.ResolvePattern(context.Background(), PathInput{Path: value}); err == nil {
			t.Errorf("expected %q to be rejected", value)
		}
	}
	regular := filepath.Join(root, "file.txt")
	writeFile(t, regular, "file")
	if _, err := service.ResolvePattern(context.Background(), PathInput{Path: filepath.Join("file.txt", "child")}); err == nil {
		t.Fatal("expected regular intermediate component to fail")
	}
	if runtime.GOOS != "windows" {
		link := filepath.Join(root, "linked")
		if err := os.Symlink(t.TempDir(), link); err != nil {
			t.Fatal(err)
		}
		if _, err := service.ResolvePattern(context.Background(), PathInput{Path: filepath.Join("linked", "future")}); err == nil {
			t.Fatal("expected symlink path to fail")
		}
	}
	if err := os.WriteFile(filepath.Join(root, ".archbase", "scope.yaml"), []byte("invalid: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ListProjectScopes(context.Background(), EmptyInput{}); err == nil {
		t.Fatal("expected invalid scope to stop listing")
	}
}

func TestServiceRejectsInvalidRootAndCancellation(t *testing.T) {
	root := t.TempDir()
	service, _ := newTestService(t, root)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.SearchPatterns(ctx, SearchPatternsInput{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	patternSource, _ := registry.NewEmbeddedSource()
	patterns, _ := registry.NewResolver(patternSource)
	ruleSource, _ := archrules.NewEmbeddedSource()
	rules, _ := archrules.NewResolver(patterns, ruleSource)
	workspaceService, _ := workspace.NewService(workspace.OSFileSystem{}, patterns)
	if _, err := NewService(filepath.Join(root, "missing"), patterns, rules, workspaceService, OSFileSystem{}); err == nil {
		t.Fatal("expected missing root to fail")
	}
	if runtime.GOOS != "windows" {
		linkedRoot := filepath.Join(t.TempDir(), "linked-root")
		if err := os.Symlink(root, linkedRoot); err != nil {
			t.Fatal(err)
		}
		if _, err := NewService(linkedRoot, patterns, rules, workspaceService, OSFileSystem{}); err == nil {
			t.Fatal("expected symlinked root to fail")
		}
		project := t.TempDir()
		linkedScope := filepath.Join(project, ".archbase")
		if err := os.Symlink(t.TempDir(), linkedScope); err != nil {
			t.Fatal(err)
		}
		linkedService, err := NewService(project, patterns, rules, workspaceService, OSFileSystem{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := linkedService.ListProjectScopes(context.Background(), EmptyInput{}); err == nil {
			t.Fatal("expected symlinked .archbase to fail")
		}
	}
}

type warningPatternSource struct {
	pattern registry.Pattern
	warning error
}

func (s warningPatternSource) Name() string { return "stale-test" }
func (s warningPatternSource) Lookup(ctx context.Context, id registry.PatternID) (registry.LookupResult, error) {
	if err := ctx.Err(); err != nil {
		return registry.LookupResult{}, err
	}
	if id != s.pattern.Entry.ID {
		return registry.LookupResult{Stale: true, Warning: s.warning}, registry.ErrPatternNotFound
	}
	return registry.LookupResult{Pattern: s.pattern, Stale: true, Warning: s.warning}, nil
}
func (s warningPatternSource) List(context.Context) (registry.ListResult, error) {
	return registry.ListResult{Entries: []registry.Entry{s.pattern.Entry}, Stale: true, Warning: s.warning}, nil
}

func TestSearchPatternsPropagatesAndDeduplicatesStaleWarnings(t *testing.T) {
	id, _ := registry.ParsePatternID("test/item@1")
	manifest := contracts.Manifest{ID: id.String(), Name: "Test Item", Description: "Searchable", Type: "pattern", Version: "1.0.0"}
	pattern := registry.Pattern{Entry: registry.Entry{ID: id}, Bundle: patterns.Bundle{Manifest: manifest}, Source: "stale-test"}
	patternsResolver, _ := registry.NewResolver(warningPatternSource{pattern: pattern, warning: errors.New("remote offline")})
	ruleSource, _ := archrules.NewEmbeddedSource()
	rulesResolver, _ := archrules.NewResolver(patternsResolver, ruleSource)
	workspaceService, _ := workspace.NewService(workspace.OSFileSystem{}, patternsResolver)
	service, err := NewService(t.TempDir(), patternsResolver, rulesResolver, workspaceService, OSFileSystem{})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.SearchPatterns(context.Background(), SearchPatternsInput{})
	if err != nil || !result.Stale || len(result.Warnings) != 1 || result.Warnings[0] != "remote offline" {
		t.Fatalf("unexpected stale result: %#v, %v", result, err)
	}
}

type testRuleSource struct {
	rules   []archrules.Rule
	listErr error
}

func (s testRuleSource) Name() string { return "test-rules" }
func (s testRuleSource) List(context.Context) (archrules.SourceListResult, error) {
	if s.listErr != nil {
		return archrules.SourceListResult{}, s.listErr
	}
	entries := make([]archrules.Entry, 0, len(s.rules))
	for _, rule := range s.rules {
		entries = append(entries, rule.Entry)
	}
	return archrules.SourceListResult{Entries: entries}, nil
}
func (s testRuleSource) Lookup(_ context.Context, id archrules.RuleID) (archrules.LookupResult, error) {
	for _, rule := range s.rules {
		if rule.Entry.ID == id {
			return archrules.LookupResult{Rule: rule}, nil
		}
	}
	return archrules.LookupResult{}, archrules.ErrRuleNotFound
}

func TestScopeRulesReturnsMultipleMatchesAndStopsOnCorruption(t *testing.T) {
	root := t.TempDir()
	patternSource, _ := registry.NewEmbeddedSource()
	patternsResolver, _ := registry.NewResolver(patternSource)
	workspaceService, _ := workspace.NewService(workspace.OSFileSystem{}, patternsResolver)
	if _, err := workspaceService.Add(context.Background(), "next/page@1234", root); err != nil {
		t.Fatal(err)
	}
	makeRule := func(rawID string) archrules.Rule {
		id, _ := archrules.ParseRuleID(rawID)
		definition := contracts.Rule{SchemaVersion: 1, ID: rawID, Name: rawID, Version: "1.0.0", Scopes: []contracts.RuleScope{{Path: "src/pages/**", Pattern: "next/page@1234"}, {Path: "src/utils/**", Pattern: "next/util@3378"}}, Rules: []string{"Keep the boundary."}}
		return archrules.Rule{Entry: archrules.Entry{ID: id, Version: "1.0.0", Source: "test-rules"}, Definition: definition, Source: "test-rules"}
	}
	source := testRuleSource{rules: []archrules.Rule{makeRule("architecture/b@1"), makeRule("architecture/a@1")}}
	rulesResolver, _ := archrules.NewResolver(patternsResolver, source)
	service, _ := NewService(root, patternsResolver, rulesResolver, workspaceService, OSFileSystem{})
	result, err := service.GetScopeRules(context.Background(), PathInput{Path: "src/pages/Home.tsx"})
	if err != nil || len(result.Rules) != 2 || result.Rules[0].ID != "architecture/a@1" || len(result.Rules[0].MatchingScopes) != 1 {
		t.Fatalf("unexpected matching rules: %#v, %v", result, err)
	}
	corruptResolver, _ := archrules.NewResolver(patternsResolver, testRuleSource{listErr: errors.New("corrupt rule index")})
	corruptService, _ := NewService(root, patternsResolver, corruptResolver, workspaceService, OSFileSystem{})
	if _, err := corruptService.GetScopeRules(context.Background(), PathInput{Path: "src/pages/Home.tsx"}); err == nil || !strings.Contains(err.Error(), "corrupt rule index") {
		t.Fatalf("expected corruption error, got %v", err)
	}
}

func TestListProjectScopesRejectsCorruptedLocalBundle(t *testing.T) {
	root := t.TempDir()
	service, workspaceService := newTestService(t, root)
	installed, err := workspaceService.Add(context.Background(), "next/page@1234", root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(installed.PatternDirectory, "Example", "Example.tsx")); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ListProjectScopes(context.Background(), EmptyInput{}); err == nil || !strings.Contains(err.Error(), "required pattern file") {
		t.Fatalf("expected corrupt bundle error, got %v", err)
	}
}

func TestMCPServerDiscoversAndCallsTypedTools(t *testing.T) {
	root := t.TempDir()
	service, workspaceService := newTestService(t, root)
	if _, err := workspaceService.Add(context.Background(), "next/page@1234", root); err != nil {
		t.Fatal(err)
	}
	server, err := NewServer(service, "test")
	if err != nil {
		t.Fatal(err)
	}
	serverTransport, clientTransport := mcpsdk.NewInMemoryTransports()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "archbase-test", Version: "test"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()
	listed, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(listed.Tools))
	for _, tool := range listed.Tools {
		names = append(names, tool.Name)
		if tool.InputSchema == nil || tool.OutputSchema == nil || tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
			t.Errorf("tool %s is missing schema or read-only annotations", tool.Name)
		}
	}
	want := []string{"get_pattern", "get_pattern_files", "get_scope_rules", "list_project_scopes", "resolve_pattern", "search_patterns"}
	sort.Strings(names)
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("unexpected tools: %v", names)
	}
	for _, call := range []struct {
		name      string
		arguments map[string]any
	}{
		{"search_patterns", map[string]any{"query": "page"}},
		{"get_pattern", map[string]any{"patternId": "next/page@1234"}},
		{"resolve_pattern", map[string]any{"path": "."}},
		{"get_pattern_files", map[string]any{"path": "."}},
		{"get_scope_rules", map[string]any{"path": "."}},
		{"list_project_scopes", map[string]any{}},
	} {
		called, err := clientSession.CallTool(ctx, &mcpsdk.CallToolParams{Name: call.name, Arguments: call.arguments})
		if err != nil || called.IsError || called.StructuredContent == nil {
			t.Fatalf("unexpected %s call: %#v, %v", call.name, called, err)
		}
	}
	failed, err := clientSession.CallTool(ctx, &mcpsdk.CallToolParams{Name: "get_pattern_files", Arguments: map[string]any{}})
	if err != nil || !failed.IsError || len(failed.Content) == 0 {
		t.Fatalf("expected a visible tool error: %#v, %v", failed, err)
	}
}

func containsFileContent(files []PatternFileContent, expected string) bool {
	for _, file := range files {
		if file.Content == expected {
			return true
		}
	}
	return false
}

func writeFile(t *testing.T, file, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
