package registry

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmbeddedSourceResolvesFoundationPattern(t *testing.T) {
	source, err := NewEmbeddedSource()
	if err != nil {
		t.Fatal(err)
	}
	id, err := ParsePatternID("next/page@1234")
	if err != nil {
		t.Fatal(err)
	}
	pattern, err := source.Lookup(id)
	if err != nil {
		t.Fatal(err)
	}
	if pattern.Manifest.Type != "pattern-bundle" || len(pattern.Manifest.Structure.Files) != 3 {
		t.Fatalf("unexpected manifest: %#v", pattern.Manifest)
	}
	if _, err := fs.ReadFile(pattern.Files, "Example/Example.tsx"); err != nil {
		t.Fatalf("pattern files are not available: %v", err)
	}
	if entries := source.List(); len(entries) != 1 || entries[0].ID != id {
		t.Fatalf("unexpected entries: %#v", entries)
	}
}

func TestLookupMissingPattern(t *testing.T) {
	source, err := NewEmbeddedSource()
	if err != nil {
		t.Fatal(err)
	}
	id, _ := ParsePatternID("next/page@9999")
	if _, err := source.Lookup(id); !errors.Is(err, ErrPatternNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestDirectorySource(t *testing.T) {
	root := createRegistry(t, "test/item@1", "test/item/1")
	source, err := NewDirectorySource(root)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := ParsePatternID("test/item@1")
	pattern, err := source.Lookup(id)
	if err != nil || !strings.HasPrefix(pattern.Source, "directory:") {
		t.Fatalf("unexpected directory lookup: %#v, %v", pattern, err)
	}
}

func TestRegistryRejectsDuplicateIDsAndTraversal(t *testing.T) {
	t.Run("duplicate", func(t *testing.T) {
		root := createRegistry(t, "test/item@1", "test/item/1")
		indexPath := filepath.Join(root, "index.yaml")
		content, err := os.ReadFile(indexPath)
		if err != nil {
			t.Fatal(err)
		}
		content = append(content, []byte("  - id: test/item@1\n    version: 1.0.0\n    path: test/item/1\n")...)
		if err := os.WriteFile(indexPath, content, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := NewDirectorySource(root); err == nil || !strings.Contains(err.Error(), "duplicate") {
			t.Fatalf("expected duplicate error, got %v", err)
		}
	})
	t.Run("traversal", func(t *testing.T) {
		root := t.TempDir()
		index := "schemaVersion: 1\npatterns:\n  - id: test/item@1\n    version: 1.0.0\n    path: ../outside\n"
		if err := os.WriteFile(filepath.Join(root, "index.yaml"), []byte(index), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := NewDirectorySource(root); err == nil || !strings.Contains(err.Error(), "escapes") {
			t.Fatalf("expected traversal error, got %v", err)
		}
	})
}

func TestGitSourceConfig(t *testing.T) {
	checkout, err := filepath.Abs(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	config := GitSourceConfig{
		URL: "https://github.com/example/registry.git", Ref: "main",
		Subdirectory: "registry", CheckoutPath: checkout,
	}
	root, err := config.DirectoryRoot()
	if err != nil || root != filepath.Join(checkout, "registry") {
		t.Fatalf("unexpected directory root: %q, %v", root, err)
	}
	config.Subdirectory = "../outside"
	if err := config.Validate(); err == nil {
		t.Fatal("expected invalid subdirectory")
	}
}

func TestPatternIDIsStrictAndCaseSensitive(t *testing.T) {
	for _, value := range []string{"Next/page@1", "next/page", "next/page@Feature"} {
		if _, err := ParsePatternID(value); err == nil {
			t.Fatalf("expected %q to be invalid", value)
		}
	}
}

func createRegistry(t *testing.T, id, registryPath string) string {
	t.Helper()
	root := t.TempDir()
	index := "schemaVersion: 1\npatterns:\n  - id: " + id + "\n    version: 1.0.0\n    path: " + registryPath + "\n"
	if err := os.WriteFile(filepath.Join(root, "index.yaml"), []byte(index), 0o600); err != nil {
		t.Fatal(err)
	}
	patternRoot := filepath.Join(root, filepath.FromSlash(registryPath))
	if err := os.MkdirAll(patternRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := "schemaVersion: 1\nid: " + id + `
name: Test item
type: pattern
version: 1.0.0
structure:
  root: "{{Name}}"
  files:
    - source: Example.txt
      destination: "{{Name}}.txt"
      required: true
allowedChanges: [content]
preserve: [file-type]
`
	if err := os.WriteFile(filepath.Join(patternRoot, "manifest.yaml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(patternRoot, "Example.txt"), []byte("example"), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}
