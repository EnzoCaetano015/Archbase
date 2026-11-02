package skillbundle

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sourceRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// macOS may expose its temporary directory through /var -> /private/var.
// Use its physical path so a system alias does not weaken symlink-rejection tests.
func tempRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func starterOptions(t *testing.T) Options {
	t.Helper()
	root := sourceRoot(t)
	return Options{Root: filepath.Join(root, "skills/archbase-architecture-starter"), Manifest: filepath.Join(root, "skills/archbase-architecture-starter/bundle.json"), Lock: filepath.Join(root, "skills/catalog.lock.json"), CatalogRoot: root}
}

func copyStarter(t *testing.T) Options {
	t.Helper()
	original := starterOptions(t)
	copied := original
	copied.Root = filepath.Join(tempRoot(t), "archbase-architecture-starter")
	if err := os.CopyFS(copied.Root, os.DirFS(original.Root)); err != nil {
		t.Fatal(err)
	}
	copied.Manifest = filepath.Join(copied.Root, "bundle.json")
	copied.Lock = filepath.Join(tempRoot(t), "catalog.lock.json")
	data, err := os.ReadFile(original.Lock)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(copied.Lock, data, 0644); err != nil {
		t.Fatal(err)
	}
	return copied
}

func rewriteJSON(t *testing.T, file string, change func(map[string]any)) {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err = json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	change(doc)
	data, err = json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(file, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestStarterDeliveryReproducibleAndExtractable(t *testing.T) {
	options := starterOptions(t)
	bundle, err := Prepare(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	first, err := bundle.Archive()
	if err != nil {
		t.Fatal(err)
	}
	second, err := bundle.Archive()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("nondeterministic ZIP")
	}
	archive, err := zip.NewReader(bytes.NewReader(first), int64(len(first)))
	if err != nil {
		t.Fatal(err)
	}
	if len(archive.File) != len(bundle.Files) {
		t.Fatal("ZIP file list differs")
	}
	previous := ""
	for _, file := range archive.File {
		if file.Name <= previous {
			t.Fatal("ZIP is not sorted")
		}
		previous = file.Name
		if file.Mode().Perm() != 0644 || file.Modified.Year() != 1980 || file.Method != zip.Store {
			t.Fatalf("unstable metadata: %s", file.Name)
		}
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		content, err := io.ReadAll(reader)
		_ = reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(content, bundle.Files[file.Name]) {
			t.Fatalf("extraction differs: %s", file.Name)
		}
		if strings.HasSuffix(file.Name, "SKILL.md") && file.Name != "SKILL.md" {
			t.Fatal("accidental second skill")
		}
	}
	destination := filepath.Join(tempRoot(t), "delivery")
	if err = bundle.Write(destination); err != nil {
		t.Fatal(err)
	}
	checksum, err := os.ReadFile(filepath.Join(destination, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(checksum), digest(first)+"  ") {
		t.Fatal("wrong checksum")
	}
	if err = bundle.Write(destination); err == nil {
		t.Fatal("existing delivery overwritten")
	}
	// Source checkout line endings must not affect the delivered bytes.
	crlf := copyStarter(t)
	err = filepath.WalkDir(crlf.Root, func(file string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		data = bytes.ReplaceAll(bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n")), []byte("\n"), []byte("\r\n"))
		return os.WriteFile(file, data, 0644)
	})
	if err != nil {
		t.Fatal(err)
	}
	converted, err := Prepare(context.Background(), crlf)
	if err != nil {
		t.Fatal(err)
	}
	convertedZIP, err := converted.Archive()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, convertedZIP) {
		t.Fatal("CRLF checkout changes ZIP")
	}
}

func TestPrepareRejectsInvalidDeliveries(t *testing.T) {
	tests := []struct {
		name, want string
		change     func(*testing.T, Options)
	}{
		{"missing-rule", "does not exist", func(t *testing.T, o Options) {
			rewriteJSON(t, o.Manifest, func(d map[string]any) { d["rules"] = []string{"architecture/missing@1"} })
		}},
		{"bad-hash", "SHA-256", func(t *testing.T, o Options) {
			rewriteJSON(t, o.Lock, func(d map[string]any) { d["files"].([]any)[0].(map[string]any)["sha256"] = strings.Repeat("0", 64) })
		}},
		{"missing-file", "missing.md", func(t *testing.T, o Options) {
			rewriteJSON(t, o.Manifest, func(d map[string]any) { d["files"] = append(d["files"].([]any), "references/missing.md") })
		}},
		{"missing-required-source", "SKILL.md", func(t *testing.T, o Options) {
			if err := os.Remove(filepath.Join(o.Root, "SKILL.md")); err != nil {
				t.Fatal(err)
			}
		}},
		{"bad-frontmatter", "frontmatter", func(t *testing.T, o Options) {
			if err := os.WriteFile(filepath.Join(o.Root, "SKILL.md"), []byte("---\nname: [invalid\n---\nInstructions long enough to be a real workflow in this isolated invalid YAML test.\n"), 0644); err != nil {
				t.Fatal(err)
			}
		}},
		{"broken-link", "missing local reference", func(t *testing.T, o Options) {
			f := filepath.Join(o.Root, "templates/orientation.md")
			data, _ := os.ReadFile(f)
			if err := os.WriteFile(f, append(data, []byte("\n[Missing](missing.md)\n")...), 0644); err != nil {
				t.Fatal(err)
			}
		}},
		{"unexpected-file", "unexpected", func(t *testing.T, o Options) {
			if err := os.WriteFile(filepath.Join(o.Root, "unlisted.txt"), []byte("unlisted"), 0644); err != nil {
				t.Fatal(err)
			}
		}},
		{"version-mismatch", "license/version", func(t *testing.T, o Options) {
			rewriteJSON(t, o.Manifest, func(d map[string]any) { d["version"] = "0.2.0" })
		}},
		{"path-traversal", "unsafe", func(t *testing.T, o Options) {
			rewriteJSON(t, o.Manifest, func(d map[string]any) { d["files"] = append(d["files"].([]any), "../secret.txt") })
		}},
		{"case-collision", "collision", func(t *testing.T, o Options) {
			rewriteJSON(t, o.Manifest, func(d map[string]any) { d["files"] = append(d["files"].([]any), "skill.md") })
		}},
		{"dropped-source-hash", "catalog lock differs", func(t *testing.T, o Options) {
			rewriteJSON(t, o.Lock, func(d map[string]any) { d["files"] = d["files"].([]any)[1:] })
		}},
		{"unknown-field", "additional properties", func(t *testing.T, o Options) {
			rewriteJSON(t, o.Manifest, func(d map[string]any) { d["unrecognized"] = true })
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := copyStarter(t)
			test.change(t, o)
			_, err := Prepare(context.Background(), o)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
		})
	}
}

func TestPortablePathAndContentPolicy(t *testing.T) {
	for _, name := range []string{"../secret", "/abs", "C:/secret", "a\\b", "a/../b", "a//b", "CON.txt", "a/NUL", "a.", "a%2fb", "a?x", "a*.md", "a\n.md", "a|b.md"} {
		if err := safePath(name); err == nil {
			t.Errorf("accepted unsafe %s", name)
		}
	}
	for _, name := range []string{".env", "scripts/run.exe", "node_modules/a.js", "other/evals/prompt.md", "go.mod", "assets/library.wasm"} {
		if err := validateText(name, []byte("text")); err == nil {
			t.Errorf("accepted forbidden %s", name)
		}
	}
	for _, content := range [][]byte{[]byte("-----BEGIN PRIVATE KEY-----\nfixture"), []byte("AKIA" + strings.Repeat("A", 16)), []byte("ghp_" + strings.Repeat("x", 36)), {0, 1, 2}, bytes.Repeat([]byte("a"), MaxFileBytes+1)} {
		if err := validateText("references/test.md", content); err == nil {
			t.Error("accepted forbidden content")
		}
	}
	files := map[string][]byte{"SKILL.md": []byte("[outside](../outside.md)")}
	if err := validateLinks(files); err == nil {
		t.Fatal("escaping link accepted")
	}
	files["SKILL.md"] = []byte("[bad](%2e%2e/outside.md)")
	if err := validateLinks(files); err == nil {
		t.Fatal("encoded escape accepted")
	}
	files["SKILL.md"] = []byte("[script](javascript:alert)")
	if err := validateLinks(files); err == nil {
		t.Fatal("unsafe scheme accepted")
	}
}

func TestRejectSymlinkedSource(t *testing.T) {
	o := copyStarter(t)
	file := filepath.Join(o.Root, "templates/orientation.md")
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(tempRoot(t), "outside.md")
	if err := os.WriteFile(outside, []byte("outside"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, file); err != nil {
		t.Skipf("symlink privilege unavailable: %v", err)
	}
	if _, err := Prepare(context.Background(), o); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink rejection: %v", err)
	}
}

func TestLockTagMustResolveToCommit(t *testing.T) {
	if _, err := CreateLock(context.Background(), sourceRoot(t), "v0.8.0", "7f6566d1b6995c41fa30ccb1d9a1d68795dca03a"); err == nil {
		t.Fatal("annotated tag object accepted as commit")
	}
}
