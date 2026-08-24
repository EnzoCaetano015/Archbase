package patterns

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"
)

func TestLoadReadsRequiredAndTracksOptionalFiles(t *testing.T) {
	fsys := fstest.MapFS{
		"pattern/manifest.yaml": &fstest.MapFile{Data: []byte(manifestWithFiles(`
    - source: Example.txt
      destination: "{{Name}}.txt"
      required: true
    - source: Optional.txt
      destination: "{{Name}}.optional.txt"
      required: false`))},
		"pattern/Example.txt": &fstest.MapFile{Data: []byte("example")},
	}
	bundle, err := Load(fsys, "pattern")
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Files) != 2 || !bundle.Files[0].Present || string(bundle.Files[0].Content) != "example" {
		t.Fatalf("unexpected required file: %#v", bundle.Files)
	}
	if bundle.Files[1].Present || bundle.Files[1].Content != nil {
		t.Fatalf("optional missing file should remain declared: %#v", bundle.Files[1])
	}
}

func TestLoadRejectsMissingRequiredFile(t *testing.T) {
	fsys := fstest.MapFS{
		"manifest.yaml": &fstest.MapFile{Data: []byte(manifestWithFiles(`
    - source: Missing.txt
      destination: "{{Name}}.txt"
      required: true`))},
	}
	_, err := Load(fsys, ".")
	if !errors.Is(err, ErrRequiredFileMissing) || !strings.Contains(err.Error(), "Missing.txt") {
		t.Fatalf("expected clear required-file error, got %v", err)
	}
}

func TestLoadRejectsDuplicateSourceAndDestination(t *testing.T) {
	tests := []struct {
		name  string
		files string
		want  error
	}{
		{
			name: "source",
			files: `
    - source: Example.txt
      destination: One.txt
      required: true
    - source: Example.txt
      destination: Two.txt
      required: true`,
			want: ErrDuplicateSource,
		},
		{
			name: "destination",
			files: `
    - source: One.txt
      destination: Example.txt
      required: true
    - source: Two.txt
      destination: Example.txt
      required: true`,
			want: ErrDuplicateDestination,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fsys := fstest.MapFS{"manifest.yaml": &fstest.MapFile{Data: []byte(manifestWithFiles(test.files))}}
			for _, name := range []string{"Example.txt", "One.txt", "Two.txt"} {
				fsys[name] = &fstest.MapFile{Data: []byte(name)}
			}
			_, err := Load(fsys, ".")
			if !errors.Is(err, test.want) {
				t.Fatalf("expected %v, got %v", test.want, err)
			}
		})
	}
}

func TestLoadRejectsDirectoryAsSource(t *testing.T) {
	fsys := fstest.MapFS{
		"manifest.yaml": &fstest.MapFile{Data: []byte(manifestWithFiles(`
    - source: Example
      destination: Example.txt
      required: true`))},
		"Example/file.txt": &fstest.MapFile{Data: []byte("content")},
	}
	_, err := Load(fsys, ".")
	if !errors.Is(err, ErrNotRegular) {
		t.Fatalf("expected regular-file error, got %v", err)
	}
}

func TestLoadRejectsTraversal(t *testing.T) {
	fsys := fstest.MapFS{
		"manifest.yaml": &fstest.MapFile{Data: []byte(strings.Replace(manifestWithFiles(`
    - source: Example.txt
      destination: Example.txt
      required: true`), "Example.txt", "../Example.txt", 1))},
	}
	if _, err := Load(fsys, "."); err == nil || !strings.Contains(err.Error(), "remain inside") {
		t.Fatalf("expected traversal error, got %v", err)
	}
}

func TestLoadRejectsSymlinkComponents(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks on Windows may require Developer Mode")
	}
	root := t.TempDir()
	outside := filepath.Join(root, "outside")
	patternRoot := filepath.Join(root, "pattern")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(patternRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "Example.txt"), []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(patternRoot, "manifest.yaml"), []byte(manifestWithFiles(`
    - source: linked/Example.txt
      destination: Example.txt
      required: true`)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(patternRoot, "linked")); err != nil {
		t.Fatal(err)
	}
	_, err := Load(os.DirFS(patternRoot), ".")
	if !errors.Is(err, ErrSymlink) {
		t.Fatalf("expected symlink error, got %v", err)
	}
}

func TestBundleDoesNotIncludeUndeclaredFiles(t *testing.T) {
	fsys := fstest.MapFS{
		"manifest.yaml": &fstest.MapFile{Data: []byte(manifestWithFiles(`
    - source: Example.txt
      destination: Example.txt
      required: true`))},
		"Example.txt": &fstest.MapFile{Data: []byte("example")},
		"README.md":   &fstest.MapFile{Data: []byte("documentation")},
	}
	bundle, err := Load(fsys, ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Files) != 1 || bundle.Files[0].Spec.Source != "Example.txt" {
		t.Fatalf("unexpected files: %#v", bundle.Files)
	}
	if _, err := fs.ReadFile(fsys, "README.md"); err != nil {
		t.Fatal(err)
	}
}

func manifestWithFiles(files string) string {
	patternType := "pattern"
	if strings.Count(files, "- source:") > 1 {
		patternType = "pattern-bundle"
	}
	return `schemaVersion: 1
id: test/example@1
name: Test pattern
type: ` + patternType + `
version: 1.0.0
structure:
  root: "{{Name}}"
  files:` + files + `
allowedChanges: [content]
preserve: [structure]
`
}
