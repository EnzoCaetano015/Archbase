package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/EnzoCaetano015/Archbase/internal/registry"
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
