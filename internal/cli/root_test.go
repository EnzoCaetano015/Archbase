package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/EnzoCaetano015/Archbase/internal/registry"
	archrules "github.com/EnzoCaetano015/Archbase/internal/rules"
	"github.com/EnzoCaetano015/Archbase/internal/workspace"
)

func TestHelpListsCurrentAndPlannedCommands(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"help"}, &stdout, &stderr, "dev")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d: %s", code, stderr.String())
	}
	for _, expected := range []string{"arc version", "arc add", "arc mcp serve"} {
		if !strings.Contains(stdout.String(), expected) {
			t.Errorf("help output does not contain %q", expected)
		}
	}
}

func TestVersion(t *testing.T) {
	for _, version := range []string{"dev", "1.2.3"} {
		t.Run(version, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Execute([]string{"version"}, &stdout, &stderr, version)
			expected := "arc " + version + "\n"
			if code != 0 || stdout.String() != expected {
				t.Fatalf("unexpected result: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
		})
	}
}

func TestUnknownCommandReturnsNonZero(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"unknown"}, &stdout, &stderr, "dev")
	if code == 0 {
		t.Fatal("expected a non-zero exit code")
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("unexpected error: %q", stderr.String())
	}
}

func TestAddResolveInspectAndCreateCommands(t *testing.T) {
	root := t.TempDir()
	for _, test := range []struct {
		args []string
		want string
	}{
		{args: []string{"add", "next/page@1234", root}, want: "Installed: next/page@1234"},
		{args: []string{"resolve", filepath.Join(root, "future")}, want: "Pattern: next/page@1234"},
		{args: []string{"inspect", root}, want: "Allowed changes:"},
		{args: []string{"create", "pages-standard", root, "--from", "next/component@4821"}, want: "Created: local/pages-standard@1"},
		{args: []string{"inspect", "next/hook@9214"}, want: "Source: official-embedded"},
	} {
		var stdout, stderr bytes.Buffer
		code := Execute(test.args, &stdout, &stderr, "dev")
		if code != 0 || !strings.Contains(stdout.String(), test.want) {
			t.Fatalf("arc %v: code=%d stdout=%q stderr=%q", test.args, code, stdout.String(), stderr.String())
		}
	}
}

func TestCommandsReturnNonZeroForMissingScopeAndCollision(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	if code := Execute([]string{"resolve", root}, &stdout, &stderr, "dev"); code == 0 || !strings.Contains(stderr.String(), "scope not found") {
		t.Fatalf("unexpected missing-scope result: code=%d stderr=%q", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"create", "example", root}, &stdout, &stderr, "dev"); code != 0 {
		t.Fatal(stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"create", "example", root}, &stdout, &stderr, "dev"); code == 0 || !strings.Contains(stderr.String(), "already exists") {
		t.Fatalf("unexpected collision result: code=%d stderr=%q", code, stderr.String())
	}
}

func TestRegistryFlagsArePassedToResolverFactory(t *testing.T) {
	var captured RegistryOptions
	dependencies := Dependencies{
		FileSystem: workspace.OSFileSystem{},
		ResolverFactory: func(_ context.Context, options RegistryOptions) (*registry.Resolver, error) {
			captured = options
			embedded, err := registry.NewEmbeddedSource()
			if err != nil {
				return nil, err
			}
			return registry.NewResolver(embedded)
		},
	}
	cache := filepath.Join(t.TempDir(), "cache")
	var stdout, stderr bytes.Buffer
	args := []string{"--registry-url", "https://example.com/registry.git", "--registry-ref", "v1", "--registry-subdir", "catalog", "--registry-cache-dir", cache, "--registry-ttl", "30m", "inspect", "next/page@1234"}
	if code := Execute(args, &stdout, &stderr, "dev", dependencies); code != 0 {
		t.Fatalf("command failed: %s", stderr.String())
	}
	if captured.URL != "https://example.com/registry.git" || captured.Ref != "v1" || captured.Subdirectory != "catalog" || captured.CacheRoot != cache || captured.TTL != 30*time.Minute {
		t.Fatalf("unexpected registry options: %#v", captured)
	}
}

func TestWarningsGoToStderr(t *testing.T) {
	var output bytes.Buffer
	printWarnings(&output, []error{context.DeadlineExceeded})
	if !strings.Contains(output.String(), "warning: context deadline exceeded") {
		t.Fatalf("unexpected warning output: %q", output.String())
	}
}

func TestRulesListInspectAndAddCommands(t *testing.T) {
	root := t.TempDir()
	for _, test := range []struct {
		args []string
		want string
	}{
		{[]string{"rules", "list"}, "architecture/next-modular@1"},
		{[]string{"rules", "inspect", "architecture/dotnet-layered@1"}, "Controllers/** -> dotnet/controller@7743"},
		{[]string{"rules", "add", "architecture/next-modular@1", "--format", "cursor", "--destination", root}, "Format: cursor"},
	} {
		var stdout, stderr bytes.Buffer
		if code := Execute(test.args, &stdout, &stderr, "dev"); code != 0 || !strings.Contains(stdout.String(), test.want) {
			t.Fatalf("arc %v: code=%d stdout=%q stderr=%q", test.args, code, stdout.String(), stderr.String())
		}
	}
	target := filepath.Join(root, ".cursor", "rules", "architecture-next-modular-1.mdc")
	if content, err := os.ReadFile(target); err != nil || !strings.Contains(string(content), "alwaysApply: false") {
		t.Fatalf("unexpected Cursor export: %q err=%v", content, err)
	}
}

func TestRulesAddValidatesArgumentsFlagsAndConflicts(t *testing.T) {
	root := t.TempDir()
	for _, args := range [][]string{
		{"rules", "add", "architecture/next-modular@1"},
		{"rules", "add", "architecture/next-modular@1", "--format", "unknown"},
		{"rules", "add", "architecture/next-modular@1", "--format", "cursor", "--merge"},
		{"rules", "add", "architecture/next-modular@1", "--format", "agents", "--overwrite"},
		{"rules", "inspect", "architecture/missing@1"},
	} {
		var stdout, stderr bytes.Buffer
		if code := Execute(args, &stdout, &stderr, "dev"); code == 0 {
			t.Fatalf("arc %v unexpectedly succeeded: %s", args, stdout.String())
		}
	}
	args := []string{"rules", "add", "architecture/next-modular@1", "--format", "copilot", "--destination", root}
	var stdout, stderr bytes.Buffer
	if code := Execute(args, &stdout, &stderr, "dev"); code != 0 {
		t.Fatal(stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute(args, &stdout, &stderr, "dev"); code == 0 || !strings.Contains(stderr.String(), "already exists") {
		t.Fatalf("expected conflict: code=%d stderr=%q", code, stderr.String())
	}
	args = append(args, "--overwrite")
	stdout.Reset()
	stderr.Reset()
	if code := Execute(args, &stdout, &stderr, "dev"); code != 0 {
		t.Fatalf("overwrite failed: %s", stderr.String())
	}
}

type cliRuleSource struct{}

func (cliRuleSource) Name() string { return "test-remote" }
func (cliRuleSource) Lookup(context.Context, archrules.RuleID) (archrules.LookupResult, error) {
	return archrules.LookupResult{}, archrules.ErrRuleNotFound
}
func (cliRuleSource) List(context.Context) (archrules.SourceListResult, error) {
	id, _ := archrules.ParseRuleID("architecture/test@1")
	return archrules.SourceListResult{
		Entries: []archrules.Entry{{ID: id, Version: "1.0.0", Source: "test-remote", Description: "Test architecture"}},
		Stale:   true, Warning: context.DeadlineExceeded,
	}, nil
}

func TestRulesCommandsPassRegistryOptionsAndPrintStaleWarnings(t *testing.T) {
	var captured RegistryOptions
	dependencies := Dependencies{
		RuleResolverFactory: func(_ context.Context, options RegistryOptions, patterns *registry.Resolver) (*archrules.Resolver, error) {
			captured = options
			return archrules.NewResolver(patterns, cliRuleSource{})
		},
	}
	cache := filepath.Join(t.TempDir(), "rules-cache")
	args := []string{"--registry-url", "https://example.com/rules.git", "--registry-ref", "v2", "--registry-subdir", "catalog", "--registry-cache-dir", cache, "--registry-ttl", "45m", "rules", "list"}
	var stdout, stderr bytes.Buffer
	if code := Execute(args, &stdout, &stderr, "dev", dependencies); code != 0 {
		t.Fatalf("rules list failed: %s", stderr.String())
	}
	if captured.URL != "https://example.com/rules.git" || captured.Ref != "v2" || captured.Subdirectory != "catalog" || captured.CacheRoot != cache || captured.TTL != 45*time.Minute {
		t.Fatalf("unexpected rule registry options: %#v", captured)
	}
	if !strings.Contains(stdout.String(), "architecture/test@1") || !strings.Contains(stderr.String(), "warning: context deadline exceeded") {
		t.Fatalf("unexpected output: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}
