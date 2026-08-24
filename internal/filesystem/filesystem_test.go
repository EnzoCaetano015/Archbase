package filesystem

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWriteFileAtomicRequiresExplicitOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "file.txt")
	if err := WriteFileAtomic(path, []byte("first"), false); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomic(path, []byte("second"), false); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	content, err := ReadFile(path)
	if err != nil || string(content) != "first" {
		t.Fatalf("existing content changed: %q, %v", content, err)
	}
	if err := WriteFileAtomic(path, []byte("second"), true); err != nil {
		t.Fatal(err)
	}
	content, err = ReadFile(path)
	if err != nil || string(content) != "second" {
		t.Fatalf("overwrite failed: %q, %v", content, err)
	}
}

func TestErrorsIncludeAffectedPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.txt")
	_, err := ReadFile(path)
	if err == nil || !strings.Contains(err.Error(), path) {
		t.Fatalf("expected path in error, got %v", err)
	}
}

func TestWriteFileAtomicDoesNotReplaceDirectories(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "existing")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomic(directory, []byte("file"), true); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("expected destination type error, got %v", err)
	}
}

func TestCopyTreeCopiesNestedFilesAndPreflightsConflicts(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	destination := filepath.Join(root, "destination")
	if err := os.MkdirAll(filepath.Join(source, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "nested", "one.txt"), []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "two.txt"), []byte("two"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CopyTree(source, destination, false); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(destination, "nested", "one.txt"))
	if err != nil || string(content) != "one" {
		t.Fatalf("nested file was not copied: %q, %v", content, err)
	}
	if err := CopyTree(source, destination, false); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestCopyTreeRejectsSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks on Windows may require Developer Mode")
	}
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "outside.txt")
	if err := os.WriteFile(target, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(source, "link.txt")); err != nil {
		t.Fatal(err)
	}
	if err := CopyTree(source, filepath.Join(root, "destination"), false); !errors.Is(err, ErrSymlink) {
		t.Fatalf("expected symlink error, got %v", err)
	}
}
