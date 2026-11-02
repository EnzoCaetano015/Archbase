package skillbundle

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/EnzoCaetano015/Archbase/internal/patterns"
)

func TestFixtureDocumentsMaterializeLogicalPaths(t *testing.T) {
	suiteFile, suite := localSuite(t)
	scenario := suite.Scenarios[1]
	files, err := loadFixture(filepath.Dir(suiteFile), scenario.Fixture)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(files["app/page.tsx"], []byte("<main>Home</main>")) || !bytes.Contains(files["features/items/components/button.tsx"], []byte(`"use client"`)) {
		t.Fatal("logical JSX content was lost")
	}
	output := filepath.Join(tempRoot(t), "project")
	if err = MaterializeFixture(suiteFile, scenario.ID, output); err != nil {
		t.Fatal(err)
	}
	for name, want := range files {
		actual, err := os.ReadFile(filepath.Join(output, filepath.FromSlash(name)))
		if err != nil || !bytes.Equal(actual, want) {
			t.Fatalf("materialized %s differs: %v", name, err)
		}
	}
	if _, err = os.Stat(filepath.Join(output, "app/page.tsx.md")); !os.IsNotExist(err) {
		t.Fatal("outer document wrapper leaked into project")
	}
	if err = MaterializeFixture(suiteFile, scenario.ID, output); err == nil {
		t.Fatal("existing project was overwritten")
	}
	if err = MaterializeFixture(suiteFile, "missing-scenario", filepath.Join(tempRoot(t), "project")); err == nil {
		t.Fatal("unknown scenario materialized")
	}
	inside := filepath.Join(filepath.Dir(suiteFile), suite.Scenarios[0].Fixture, "generated")
	if err = MaterializeFixture(suiteFile, scenario.ID, inside); err == nil {
		t.Fatal("materialization wrote inside a versioned fixture")
	}
	if _, err = os.Stat(inside); !os.IsNotExist(err) {
		t.Fatal("rejected materialization modified fixture")
	}
}

func TestFixtureDigestUsesDecodedCode(t *testing.T) {
	root := tempRoot(t)
	logical := map[string][]byte{
		"app/page.tsx":                           []byte("export default function Page() { return <main /> }\n"),
		"package.json":                           []byte("{\"private\": true}\n"),
		"AGENTS.md":                              []byte("# Example project policy\n"),
		".archbase/patterns/page/example.tsx.md": []byte("```tsx\nexport const Page = () => <main />\n```\n"),
	}
	for name, code := range logical {
		if err := writeNew(filepath.Join(root, "fixture", filepath.FromSlash(name)+".md"), patterns.EncodeDocument(name, code)); err != nil {
			t.Fatal(err)
		}
	}
	files, err := loadFixture(root, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range logical {
		if !bytes.Equal(files[name], want) {
			t.Fatalf("logical content changed: %s", name)
		}
	}
	before, err := fixtureDigest(root, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "fixture/app/page.tsx.md")
	// Changing Markdown delimiters and CRLF does not change the evaluated code.
	document := "````tsx\r\n" + strings.ReplaceAll(string(logical["app/page.tsx"]), "\n", "\r\n") + "````\r\n"
	if err = os.WriteFile(file, []byte(document), 0644); err != nil {
		t.Fatal(err)
	}
	after, err := fixtureDigest(root, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatal("Markdown presentation changed the logical fixture revision")
	}
}

func TestFixtureRejectsRawCodeAndInvalidDocuments(t *testing.T) {
	for _, test := range []struct{ name, document string }{
		{"app/page.tsx", "export default function Page() { return <main/> }\n"},
		{"app/page.tsx.md", "```json\n{}\n```\n"},
		{"app/page.tsx.md", "```tsx\nexport const Page = () => <main/>\n"},
	} {
		t.Run(test.name+test.document[:3], func(t *testing.T) {
			root := tempRoot(t)
			if err := writeNew(filepath.Join(root, "fixture", filepath.FromSlash(test.name)), []byte(test.document)); err != nil {
				t.Fatal(err)
			}
			if _, err := loadFixture(root, "fixture"); err == nil {
				t.Fatal("invalid fixture source accepted")
			}
		})
	}
}
