package diagnostics

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestWindowsJunctionRootAndBundleRejected(t *testing.T) {
	root := tempDir(t)
	outside := tempDir(t)
	link := filepath.Join(root, "junction")
	if output, err := exec.Command("cmd", "/c", "mklink", "/J", link, outside).CombinedOutput(); err != nil {
		t.Skipf("junction creation unavailable: %v %s", err, output)
	}
	if r, err := OpenRoot(context.Background(), link, DefaultLimits); err == nil {
		r.Close()
		t.Fatal("junction root accepted")
	}
	r, err := OpenRoot(context.Background(), root, DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if _, err = r.ReadFile("junction/secret.txt"); err == nil {
		t.Fatal("junction component accepted")
	}
}

func TestWindowsJunctionSwapCannotReadOutside(t *testing.T) {
	root := tempDir(t)
	outside := tempDir(t)
	put(t, root, "bundle/file.md", []byte("original"))
	put(t, outside, "file.md", []byte("OUTSIDE_SENTINEL"))
	r, err := OpenRoot(context.Background(), root, DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	r.(*rootReader).beforeOpen = func(name string) {
		if name != "bundle/file.md" {
			return
		}
		if err := os.Rename(filepath.Join(root, "bundle"), filepath.Join(root, "saved")); err != nil {
			t.Fatal(err)
		}
		if output, err := exec.Command("cmd", "/c", "mklink", "/J", filepath.Join(root, "bundle"), outside).CombinedOutput(); err != nil {
			t.Fatalf("junction creation failed: %v %s", err, output)
		}
	}
	if data, err := r.ReadFile("bundle/file.md"); err == nil || len(data) > 0 {
		t.Fatalf("junction swap returned outside bytes: %q %v", data, err)
	}
}
