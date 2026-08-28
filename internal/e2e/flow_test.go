package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/EnzoCaetano015/Archbase/internal/cli"
	archmcp "github.com/EnzoCaetano015/Archbase/internal/mcp"
	"github.com/EnzoCaetano015/Archbase/internal/registry"
	archrules "github.com/EnzoCaetano015/Archbase/internal/rules"
	"github.com/EnzoCaetano015/Archbase/internal/workspace"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestPrimaryFlowNextAndDotnet(t *testing.T) {
	remote := createGitRegistry(t)
	for _, scenario := range []struct {
		name          string
		fixture       string
		rootPattern   string
		nested        string
		localName     string
		customFile    string
		target        string
		outsideTarget string
		ruleID        string
		exported      string
	}{
		{
			name: "next", fixture: "next", rootPattern: "next/page@1234", nested: "src/pages", localName: "pages-standard",
			customFile: "Example/Example.tsx", target: "src/pages/Admin/Page.tsx", outsideTarget: "src/components/Card.tsx",
			ruleID: "architecture/next-modular@1", exported: ".cursor/rules/architecture-next-modular-1.mdc",
		},
		{
			name: "dotnet", fixture: "dotnet", rootPattern: "dotnet/controller@7743", nested: "src/Controllers", localName: "controllers-standard",
			customFile: "ExampleController.cs", target: "src/Controllers/AdminController.cs", outsideTarget: "src/Services/ExampleService.cs",
			ruleID: "architecture/dotnet-layered@1", exported: ".cursor/rules/architecture-dotnet-layered-1.mdc",
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			project := filepath.Join(t.TempDir(), "project")
			copyTree(t, filepath.Join("testdata", scenario.fixture), project)
			cache := filepath.Join(t.TempDir(), "cache")
			global := []string{"--registry-url", fileURL(remote), "--registry-ref", "main", "--registry-cache-dir", cache}
			runCLI(t, append(global, "add", scenario.rootPattern, project)...)
			nested := filepath.Join(project, filepath.FromSlash(scenario.nested))
			runCLI(t, append(global, "create", scenario.localName, nested, "--from", scenario.rootPattern)...)

			custom := "customized " + scenario.name + " bundle\n"
			customPath := filepath.Join(nested, ".archbase", "patterns", scenario.localName, filepath.FromSlash(scenario.customFile))
			if err := os.WriteFile(customPath, []byte(custom), 0o644); err != nil {
				t.Fatal(err)
			}
			resolvedNested := runCLI(t, append(global, "resolve", filepath.Join(project, filepath.FromSlash(scenario.target)))...)
			if !strings.Contains(resolvedNested, "Pattern: local/"+scenario.localName+"@1") {
				t.Fatalf("nearest scope did not win:\n%s", resolvedNested)
			}
			resolvedParent := runCLI(t, append(global, "resolve", filepath.Join(project, filepath.FromSlash(scenario.outsideTarget)))...)
			if !strings.Contains(resolvedParent, "Pattern: "+scenario.rootPattern) {
				t.Fatalf("parent scope was not preserved:\n%s", resolvedParent)
			}
			runCLI(t, append(global, "rules", "add", scenario.ruleID, "--format", "cursor", "--destination", project)...)
			if _, err := os.Stat(filepath.Join(project, filepath.FromSlash(scenario.exported))); err != nil {
				t.Fatalf("rule export missing: %v", err)
			}

			service := newGitMCPService(t, project, remote, cache)
			client, closeClient := connectMCP(t, service)
			defer closeClient()
			resolved := callTool[archmcp.ResolvedPatternOutput](t, client, "resolve_pattern", map[string]any{"path": scenario.target})
			if resolved.Manifest.ID != "local/"+scenario.localName+"@1" || resolved.Origin == nil || !strings.HasPrefix(resolved.Origin.Registry, "git:") {
				t.Fatalf("MCP resolved %q", resolved.Manifest.ID)
			}
			files := callTool[archmcp.PatternFilesOutput](t, client, "get_pattern_files", map[string]any{"path": scenario.target})
			if !hasContent(files, custom) {
				t.Fatalf("MCP did not return local customization: %#v", files.Files)
			}
			rules := callTool[archmcp.ScopeRulesOutput](t, client, "get_scope_rules", map[string]any{"path": scenario.target})
			if rules.OriginID != scenario.rootPattern || len(rules.Rules) != 1 || rules.Rules[0].ID != scenario.ruleID {
				t.Fatalf("unexpected MCP rules: %#v", rules)
			}
			scopes := callTool[archmcp.ProjectScopesOutput](t, client, "list_project_scopes", map[string]any{})
			if len(scopes.Scopes) != 2 || scopes.Scopes[0].Path != "." || scopes.Scopes[1].Path != scenario.nested {
				t.Fatalf("unexpected MCP scopes: %#v", scopes.Scopes)
			}
		})
	}
}

func newGitMCPService(t *testing.T, project, remote, cache string) *archmcp.Service {
	t.Helper()
	config := registry.GitSourceConfig{URL: fileURL(remote), Ref: "main", CacheRoot: cache, TTL: 15 * time.Minute}
	gitPatterns, err := registry.NewGitSource(config)
	if err != nil {
		t.Fatal(err)
	}
	embeddedPatterns, _ := registry.NewEmbeddedSource()
	patterns, _ := registry.NewResolver(gitPatterns, embeddedPatterns)
	gitRules, err := archrules.NewGitSource(config)
	if err != nil {
		t.Fatal(err)
	}
	embeddedRules, _ := archrules.NewEmbeddedSource()
	rules, _ := archrules.NewResolver(patterns, gitRules, embeddedRules)
	workspaceService, _ := workspace.NewService(workspace.OSFileSystem{}, patterns)
	service, err := archmcp.NewService(project, patterns, rules, workspaceService, archmcp.OSFileSystem{})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func connectMCP(t *testing.T, service *archmcp.Service) (*mcpsdk.ClientSession, func()) {
	t.Helper()
	server, err := archmcp.NewServer(service, "e2e")
	if err != nil {
		t.Fatal(err)
	}
	serverTransport, clientTransport := mcpsdk.NewInMemoryTransports()
	ctx, cancel := context.WithCancel(context.Background())
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "archbase-e2e", Version: "test"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		serverSession.Close()
		cancel()
		t.Fatal(err)
	}
	return clientSession, func() {
		clientSession.Close()
		serverSession.Close()
		cancel()
	}
}

func callTool[T any](t *testing.T, client *mcpsdk.ClientSession, name string, arguments map[string]any) T {
	t.Helper()
	result, err := client.CallTool(context.Background(), &mcpsdk.CallToolParams{Name: name, Arguments: arguments})
	if err != nil || result.IsError {
		t.Fatalf("call %s failed: result=%#v err=%v", name, result, err)
	}
	data, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var output T
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatal(err)
	}
	return output
}

func hasContent(files archmcp.PatternFilesOutput, expected string) bool {
	for _, file := range files.Files {
		if file.Content == expected {
			return true
		}
	}
	return false
}

func runCLI(t *testing.T, args ...string) string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if code := cli.Execute(args, &stdout, &stderr, "e2e"); code != 0 {
		t.Fatalf("arc %v failed: %s", args, stderr.String())
	}
	return stdout.String()
}

func createGitRegistry(t *testing.T) string {
	t.Helper()
	_, currentFile, _, _ := runtime.Caller(0)
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	remote := filepath.Join(t.TempDir(), "registry")
	copyTree(t, filepath.Join(repositoryRoot, "registry"), remote)
	repository, err := git.PlainInit(remote, false)
	if err != nil {
		t.Fatal(err)
	}
	worktree, err := repository.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	if err := worktree.AddWithOptions(&git.AddOptions{All: true}); err != nil {
		t.Fatal(err)
	}
	hash, err := worktree.Commit("e2e registry", &git.CommitOptions{Author: &object.Signature{Name: "Archbase E2E", Email: "e2e@archbase.local", When: time.Unix(1, 0).UTC()}})
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
	return remote
}

func copyTree(t *testing.T, source, destination string) {
	t.Helper()
	if err := filepath.WalkDir(source, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, current)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(current)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	}); err != nil {
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
