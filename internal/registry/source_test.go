package registry

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	registrydata "github.com/EnzoCaetano015/Archbase/registry"
)

func TestEmbeddedSourceContainsAllFoundationPatterns(t *testing.T) {
	source, err := NewEmbeddedSource()
	if err != nil {
		t.Fatal(err)
	}
	listed, err := source.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{
		"dotnet/controller@7743",
		"dotnet/repository@5532",
		"dotnet/service@1172",
		"next/component@4821",
		"next/hook@9214",
		"next/page@1234",
		"next/util@3378",
	}
	if len(listed.Entries) != len(expected) {
		t.Fatalf("expected %d entries, got %#v", len(expected), listed.Entries)
	}
	for index, rawID := range expected {
		if listed.Entries[index].ID.String() != rawID {
			t.Fatalf("entry %d is %q, expected %q", index, listed.Entries[index].ID, rawID)
		}
		result, err := source.Lookup(context.Background(), listed.Entries[index].ID)
		if err != nil {
			t.Fatalf("lookup %s: %v", rawID, err)
		}
		if result.Pattern.Bundle.Manifest.ID != rawID {
			t.Fatalf("manifest mismatch for %s", rawID)
		}
		for _, file := range result.Pattern.Bundle.Files {
			if file.Spec.Required && !file.Present {
				t.Fatalf("required file %s is absent from %s", file.Spec.Source, rawID)
			}
		}
		readme := filepath.ToSlash(filepath.Join(listed.Entries[index].Path, "README.md"))
		if _, err := fs.ReadFile(registrydata.FS, readme); err != nil {
			t.Fatalf("README missing for %s: %v", rawID, err)
		}
	}
}

func TestLookupMissingPattern(t *testing.T) {
	source, err := NewEmbeddedSource()
	if err != nil {
		t.Fatal(err)
	}
	id, _ := ParsePatternID("next/page@9999")
	if _, err := source.Lookup(context.Background(), id); !errors.Is(err, ErrPatternNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestDirectorySource(t *testing.T) {
	root := createRegistry(t, "test/item@1", "test/item/1", "first")
	source, err := NewDirectorySource(root)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := ParsePatternID("test/item@1")
	result, err := source.Lookup(context.Background(), id)
	if err != nil || !strings.HasPrefix(result.Pattern.Source, "directory:") {
		t.Fatalf("unexpected directory lookup: %#v, %v", result, err)
	}
	if string(result.Pattern.Bundle.Files[0].Content) != "first" {
		t.Fatalf("unexpected bundle content: %#v", result.Pattern.Bundle.Files)
	}
}

func TestRegistryRejectsDuplicateUnsortedTraversalAndCorruption(t *testing.T) {
	t.Run("duplicate", func(t *testing.T) {
		root := createRegistry(t, "test/item@1", "test/item/1", "example")
		appendIndex(t, root, "  - id: test/item@1\n    version: 1.0.0\n    path: test/item/1\n")
		if _, err := NewDirectorySource(root); err == nil || !strings.Contains(err.Error(), "duplicate") {
			t.Fatalf("expected duplicate error, got %v", err)
		}
	})
	t.Run("unsorted", func(t *testing.T) {
		root := createRegistry(t, "test/zeta@1", "test/zeta/1", "zeta")
		secondRoot := filepath.Join(root, "test", "alpha", "1")
		writePattern(t, secondRoot, "test/alpha@1", "alpha")
		appendIndex(t, root, "  - id: test/alpha@1\n    version: 1.0.0\n    path: test/alpha/1\n")
		if _, err := NewDirectorySource(root); err == nil || !strings.Contains(err.Error(), "sorted") {
			t.Fatalf("expected sorted-index error, got %v", err)
		}
	})
	t.Run("traversal", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "index.yaml"), "schemaVersion: 1\npatterns:\n  - id: test/item@1\n    version: 1.0.0\n    path: ../outside\n")
		if _, err := NewDirectorySource(root); err == nil || !strings.Contains(err.Error(), "escapes") {
			t.Fatalf("expected traversal error, got %v", err)
		}
	})
	t.Run("missing required file", func(t *testing.T) {
		root := createRegistry(t, "test/item@1", "test/item/1", "example")
		if err := os.Remove(filepath.Join(root, "test", "item", "1", "Example.txt")); err != nil {
			t.Fatal(err)
		}
		if _, err := NewDirectorySource(root); err == nil || !strings.Contains(err.Error(), "required pattern file") {
			t.Fatalf("expected required-file error, got %v", err)
		}
	})
}

func TestSourceHonorsCanceledContext(t *testing.T) {
	source, err := NewEmbeddedSource()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	id, _ := ParsePatternID("next/page@1234")
	if _, err := source.Lookup(ctx, id); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

func TestDirectorySourceRejectsSymlinkRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks on Windows may require Developer Mode")
	}
	root := createRegistry(t, "test/item@1", "test/item/1", "example")
	link := filepath.Join(t.TempDir(), "registry-link")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	if _, err := NewDirectorySource(link); err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("expected symlink root rejection, got %v", err)
	}
}

func TestPatternIDIsStrictAndCaseSensitive(t *testing.T) {
	for _, value := range []string{"Next/page@1", "next/page", "next/page@Feature"} {
		if _, err := ParsePatternID(value); err == nil {
			t.Fatalf("expected %q to be invalid", value)
		}
	}
}

func createRegistry(t *testing.T, id, registryPath, content string) string {
	t.Helper()
	root := t.TempDir()
	index := "schemaVersion: 1\npatterns:\n  - id: " + id + "\n    version: 1.0.0\n    path: " + registryPath + "\n"
	writeFile(t, filepath.Join(root, "index.yaml"), index)
	writePattern(t, filepath.Join(root, filepath.FromSlash(registryPath)), id, content)
	return root
}

func writePattern(t *testing.T, root, id, content string) {
	t.Helper()
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
	writeFile(t, filepath.Join(root, "manifest.yaml"), manifest)
	writeFile(t, filepath.Join(root, "Example.txt"), content)
	writeFile(t, filepath.Join(root, "README.md"), "# Test pattern\n")
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func appendIndex(t *testing.T, root, content string) {
	t.Helper()
	path := filepath.Join(root, "index.yaml")
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := file.WriteString(content); err != nil {
		t.Fatal(err)
	}
}
