//go:build !windows

package diagnostics

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestUnixDeniedFileRemainsUnassessed(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission bits")
	}
	root := tempDir(t)
	put(t, root, "AGENTS.md", []byte("external instructions"))
	name := filepath.Join(root, "AGENTS.md")
	if err := os.Chmod(name, 0); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(name, 0600)
	r := Run(context.Background(), root, "dev", Options{})
	if r.ExitCode() != 3 || check(t, r, "managed-indices").Reason != "permission-denied" {
		t.Fatalf("%+v", r)
	}
}

func TestUnixFIFOSwapDoesNotBlock(t *testing.T) {
	root := tempDir(t)
	put(t, root, "file.md", []byte("original"))
	r, err := OpenRoot(context.Background(), root, DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	r.(*rootReader).beforeOpen = func(name string) {
		if err := os.Remove(filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
		if err := syscall.Mkfifo(filepath.Join(root, name), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if data, err := r.ReadFile("file.md"); err == nil || len(data) > 0 {
		t.Fatalf("FIFO was not rejected: %q %v", data, err)
	}
}
