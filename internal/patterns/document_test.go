package patterns

import (
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/EnzoCaetano015/Archbase/internal/contracts"
)

func TestDocumentRoundTrip(t *testing.T) {
	for _, name := range []string{"file.ts", "file.tsx", "file.cs", "file.py", "file.java", "file.dart", "file.astro", "file.yaml", "file.md", "file.unknown"} {
		code := []byte("// Olá 世界\nconst backticks = \"```\";\n")
		doc := EncodeDocument(name, code)
		actual, lang, err := DecodeDocument(name+".md", doc)
		if err != nil || string(actual) != string(code) || lang != LanguageFor(name) {
			t.Fatalf("%s: %q %q %v", name, actual, lang, err)
		}
	}
}

func TestDartDocumentLanguage(t *testing.T) {
	code := []byte("class Example {}\n")
	document := EncodeDocument("example.dart", code)
	if !strings.HasPrefix(string(document), "```dart\n") {
		t.Fatalf("Dart document has the wrong language: %s", document)
	}
	decoded, language, err := DecodeDocument("example.dart.md", document)
	if err != nil || language != "dart" || string(decoded) != string(code) {
		t.Fatalf("Dart extraction: %q %s %v", decoded, language, err)
	}
	if _, _, err := DecodeDocument("example.dart.md", []byte("```text\nclass Example {}\n```\n")); err == nil {
		t.Fatal("accepted a Dart document with the wrong language")
	}
}
func TestRejectMalformedMarkdown(t *testing.T) {
	for _, doc := range []string{"plain", "```typescript\ncode", "```python\ncode\n```", "before\n```typescript\ncode\n```", "```typescript\ncode\n```\nafter", "```typescript\none\n```\n```typescript\ntwo\n```", "```typescript\n\xff\n```", "```\ncode\n```"} {
		if _, _, err := DecodeDocument("file.ts.md", []byte(doc)); err == nil {
			t.Fatalf("accepted %q", doc)
		}
	}
	code, lang, err := DecodeDocument("file.ts.md", []byte("\n```ts\nconst x=1\n```\n"))
	if err != nil || lang != "typescript" || string(code) != "const x=1\n" {
		t.Fatalf("alias: %q %s %v", code, lang, err)
	}
}
func TestBundlePreservesDocumentAndBinary(t *testing.T) {
	spec := contracts.PatternFile{Source: "example.ts", Destination: "{{Name}}.ts", Required: true}
	manifest := contracts.Manifest{SchemaVersion: 1, ID: "test/example@1", Name: "example", Version: "1.0.0", Type: "pattern", Structure: contracts.PatternStructure{Root: ".", Files: []contracts.PatternFile{spec}}, AllowedChanges: []string{"content"}, Preserve: []string{"structure"}}
	original := Bundle{Manifest: manifest, Files: []File{{Spec: spec, Present: true, Content: []byte("const x = 1;\n")}}}
	converted, err := DocumentBundle(original)
	if err != nil {
		t.Fatal(err)
	}
	if original.Manifest.Structure.Files[0].Source != "example.ts" || converted.Files[0].Spec.Source != "example.ts.md" {
		t.Fatal("source mutation")
	}
	fsys := fstest.MapFS{"manifest.yaml": {Data: converted.ManifestContent}, "example.ts.md": {Data: converted.Files[0].DocumentContent}}
	loaded, err := Load(fsys, ".")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(loaded.Files[0].Content), "```") || loaded.Files[0].Language != "typescript" {
		t.Fatal(loaded)
	}
	original.Files[0].Content = []byte{0, 1, 255}
	binary, err := DocumentBundle(original)
	if err != nil || binary.Files[0].Spec.Source != "example.ts" || binary.Files[0].Spec.Format != "" {
		t.Fatalf("binary %v", err)
	}
}

func TestMarkdownLoaderErrorsRetainFieldAndPath(t *testing.T) {
	f := contracts.PatternFile{Source: "example.ts.md", Destination: "example.ts", Required: true, Format: "markdown-code"}
	m := contracts.Manifest{SchemaVersion: 1, ID: "test/example@1", Name: "Example", Version: "1.0.0", Type: "pattern", Structure: contracts.PatternStructure{Root: ".", Files: []contracts.PatternFile{f}}, AllowedChanges: []string{"content"}, Preserve: []string{"structure"}}
	data, err := contracts.EncodeManifest(m)
	if err != nil {
		t.Fatal(err)
	}
	fsys := fstest.MapFS{"manifest.yaml": {Data: data}, "example.ts.md": {Data: []byte("invalid document")}}
	_, err = Load(fsys, ".")
	var structured *Error
	if !errors.As(err, &structured) || structured.Manifest != "manifest.yaml" || structured.Path != "example.ts.md" || structured.Field != "/structure/files/0/source" {
		t.Fatalf("lost error context: %v", err)
	}
	m.Structure.Files[0].Required = false
	data, err = contracts.EncodeManifest(m)
	if err != nil {
		t.Fatal(err)
	}
	fsys = fstest.MapFS{"manifest.yaml": {Data: data}}
	loaded, err := Load(fsys, ".")
	if err != nil || loaded.Files[0].Present || loaded.Files[0].Spec.Format != "markdown-code" {
		t.Fatalf("missing optional: %#v %v", loaded, err)
	}
}
