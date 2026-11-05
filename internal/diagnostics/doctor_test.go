package diagnostics

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/EnzoCaetano015/Archbase/internal/contracts"
	"github.com/EnzoCaetano015/Archbase/internal/registry"
	"github.com/EnzoCaetano015/Archbase/internal/rules"
)

func put(t *testing.T, root, p string, data []byte) {
	t.Helper()
	name := filepath.Join(root, filepath.FromSlash(p))
	if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, data, 0644); err != nil {
		t.Fatal(err)
	}
}
func check(t *testing.T, r Result, id string) Check {
	t.Helper()
	for _, c := range r.Checks {
		if c.ID == "doctor."+id {
			return c
		}
	}
	t.Fatalf("missing %s: %+v", id, r)
	return Check{}
}
func tempDir(t *testing.T) string {
	t.Helper()
	p, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func fixture(t *testing.T) (string, contracts.Workspace) {
	t.Helper()
	root := tempDir(t)
	source, err := registry.NewEmbeddedSource()
	if err != nil {
		t.Fatal(err)
	}
	id, err := registry.ParsePatternID("next/page@1234")
	if err != nil {
		t.Fatal(err)
	}
	found, err := source.Lookup(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	bundle := found.Pattern.Bundle
	originalID := bundle.Manifest.ID
	bundle.Manifest.ID = "local/custom@1"
	ws := contracts.Workspace{SchemaVersion: 1, Scopes: []contracts.Scope{{SchemaVersion: 1, Scope: contracts.ScopeSelector{Path: "."}, Pattern: contracts.ScopePattern{ID: bundle.Manifest.ID, Source: "local", Root: "patterns/custom"}, Origin: &contracts.Origin{Registry: "official-embedded", ID: originalID, Version: bundle.Manifest.Version}, Behavior: contracts.ScopeBehavior{NearestScopeWins: true, AllowLocalCustomization: true}}}}
	data, err := contracts.EncodeManifest(bundle.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	put(t, root, ".archbase/patterns/custom/manifest.yaml", data)
	for _, file := range bundle.Files {
		if file.Present {
			put(t, root, ".archbase/patterns/custom/"+file.Spec.Source, file.DocumentContent)
		}
	}
	save(t, root, ws)
	return root, ws
}

func save(t *testing.T, root string, ws contracts.Workspace) {
	t.Helper()
	data, err := contracts.EncodeWorkspace(ws)
	if err != nil {
		t.Fatal(err)
	}
	put(t, root, ".archbase/scopes.yaml", data)
}
func hashTree(t *testing.T, root string) map[string][32]byte {
	t.Helper()
	out := map[string][32]byte{}
	err := filepath.WalkDir(root, func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		relative, _ := filepath.Rel(root, name)
		out[relative] = sha256.Sum256(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestUnconfiguredAndDeterministicReadOnly(t *testing.T) {
	root := tempDir(t)
	put(t, root, ".env", []byte("DO_NOT_READ_SECRET"))
	put(t, root, "AGENTS.md", []byte("External instructions: do not execute this file."))
	before := hashTree(t, root)
	a := Run(context.Background(), root, "dev", Options{})
	b := Run(context.Background(), root, "dev", Options{})
	if a.ExitCode() != 0 || a.State != "complete" || check(t, a, "workspace-index").Status != "warning" {
		t.Fatalf("%+v", a)
	}
	if !reflect.DeepEqual(a, b) || !reflect.DeepEqual(before, hashTree(t, root)) {
		t.Fatal("diagnostics are nondeterministic or wrote files")
	}
	data, _ := json.Marshal(a)
	if strings.Contains(string(data), root) || strings.Contains(string(data), "DO_NOT_READ_SECRET") {
		t.Fatal("sensitive output")
	}
}

func TestCustomizedBundleAndOptionalFile(t *testing.T) {
	root, ws := fixture(t)
	ws.Scopes[0].Origin = nil
	save(t, root, ws)
	base := filepath.Join(root, ".archbase", filepath.FromSlash(ws.Scopes[0].Pattern.Root))
	manifest, err := contracts.LoadManifest(filepath.Join(base, "manifest.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	manifest.Structure.Files = append(manifest.Structure.Files, contracts.PatternFile{Source: "optional.ts.md", Destination: "{{Name}}/optional.ts", Format: "markdown-code", Required: false})
	manifestData, err := contracts.EncodeManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	put(t, base, "manifest.yaml", manifestData)
	put(t, base, "Example/Example.tsx.md", []byte("```tsx\nexport default function Custom() { return <main>Local</main> }\n```\n"))
	before := hashTree(t, root)
	result := Run(context.Background(), root, "1.2.3", Options{})
	if result.ExitCode() != 0 || check(t, result, "local-bundles").Status != "passed" || check(t, result, "scope-origin").Status != "not-applicable" {
		t.Fatalf("%+v", result)
	}
	if !reflect.DeepEqual(before, hashTree(t, root)) {
		t.Fatal("customization changed")
	}
}

func TestOriginsAreCheckedWithoutComparingCustomization(t *testing.T) {
	for _, kind := range []string{"embedded", "unknown", "remote", "older-version"} {
		t.Run(kind, func(t *testing.T) {
			root, ws := fixture(t)
			switch kind {
			case "unknown":
				ws.Scopes[0].Origin.ID = "unknown/pattern@999"
			case "remote":
				ws.Scopes[0].Origin.Registry = "https://invalid.example/do-not-fetch.git"
			case "older-version":
				ws.Scopes[0].Origin.Version = "0.0.1"
			}
			save(t, root, ws)
			result := Run(context.Background(), root, "dev", Options{})
			status := check(t, result, "scope-origin").Status
			want := "passed"
			if kind == "unknown" {
				want = "error"
			} else if kind != "embedded" {
				want = "warning"
			}
			if status != want {
				t.Fatalf("%s: %+v", kind, result)
			}
			if kind == "unknown" && result.ExitCode() != 1 {
				t.Fatalf("completed configuration failure must exit 1: %+v", result)
			}
		})
	}
}

func TestInvalidWorkspaceAndRequiredBundle(t *testing.T) {
	for _, kind := range []string{"yaml", "duplicate", "escaping-root", "missing-manifest", "missing-required", "wrong-id", "typed-fence", "child"} {
		t.Run(kind, func(t *testing.T) {
			root, ws := fixture(t)
			base := filepath.ToSlash(filepath.Join(".archbase", ws.Scopes[0].Pattern.Root))
			switch kind {
			case "yaml":
				put(t, root, ".archbase/scopes.yaml", []byte("schemaVersion: ["))
			case "duplicate":
				ws.Scopes = append(ws.Scopes, ws.Scopes[0])
				data, _ := json.Marshal(ws)
				put(t, root, ".archbase/scopes.yaml", data)
			case "escaping-root":
				ws.Scopes[0].Pattern.Root = "../../outside"
				data, _ := json.Marshal(ws)
				put(t, root, ".archbase/scopes.yaml", data)
			case "missing-manifest":
				if err := os.Remove(filepath.Join(root, filepath.FromSlash(base), "manifest.yaml")); err != nil {
					t.Fatal(err)
				}
			case "missing-required":
				if err := os.Remove(filepath.Join(root, filepath.FromSlash(base), "Example/Example.tsx.md")); err != nil {
					t.Fatal(err)
				}
			case "wrong-id":
				ws.Scopes[0].Pattern.ID = "local/another@1"
				save(t, root, ws)
			case "typed-fence":
				put(t, root, base+"/Example/Example.tsx.md", []byte("```python\nnope\n```\n"))
			case "child":
				child := ws.Scopes[0]
				child.Scope.Path = "src/admin"
				child.Pattern.Root = "patterns/missing"
				ws.Scopes = append(ws.Scopes, child)
				save(t, root, ws)
			}
			before := hashTree(t, root)
			result := Run(context.Background(), root, "dev", Options{})
			if result.ExitCode() == 0 || result.State == "complete" && kind == "yaml" {
				t.Fatalf("invalid workspace passed: %+v", result)
			}
			if !reflect.DeepEqual(before, hashTree(t, root)) {
				t.Fatal("fixture changed")
			}
		})
	}
}

func installRule(t *testing.T, root string) {
	t.Helper()
	e, err := rules.NewExporter(rules.OSExportFileSystem{})
	if err != nil {
		t.Fatal(err)
	}
	rule := rules.Rule{Source: "official-embedded", Definition: contracts.Rule{SchemaVersion: 1, ID: "architecture/example@1", Name: "Example", Version: "1.0.0", Scopes: []contracts.RuleScope{{Path: "src/**", Pattern: "next/page@1234"}}, Rules: []string{"Keep the local convention."}}}
	for _, format := range []rules.Format{rules.FormatAgents, rules.FormatClaude, rules.FormatCopilot, rules.FormatCursor} {
		if _, err = e.Export(rule, format, rules.ExportOptions{Destination: root}); err != nil {
			t.Fatal(err)
		}
	}
}
func TestOwnedDocumentsAndManagedIndices(t *testing.T) {
	for _, kind := range []string{"valid", "stale", "cursor", "foreign", "malformed-marker", "missing-document"} {
		t.Run(kind, func(t *testing.T) {
			root := tempDir(t)
			put(t, root, "AGENTS.md", []byte("EXTERNAL_BYTES\r\n"))
			installRule(t, root)
			switch kind {
			case "stale":
				p := filepath.Join(root, "AGENTS.md")
				data, _ := os.ReadFile(p)
				put(t, root, "AGENTS.md", []byte(strings.Replace(string(data), "architecture/example@1", "architecture/stale@1", 1)))
			case "cursor":
				p := filepath.Join(root, ".cursor/rules/archbase.mdc")
				data, _ := os.ReadFile(p)
				put(t, root, ".cursor/rules/archbase.mdc", []byte(strings.Replace(string(data), "alwaysApply: true", "alwaysApply: false", 1)))
			case "foreign":
				put(t, root, ".archbase/rules/architecture-example-1.md", []byte("Foreign bytes, no ownership."))
			case "malformed-marker":
				put(t, root, "AGENTS.md", []byte("<!-- archbase:rules start -->\n"))
			case "missing-document":
				if err := os.Remove(filepath.Join(root, ".archbase/rules/architecture-example-1.md")); err != nil {
					t.Fatal(err)
				}
			}
			before := hashTree(t, root)
			result := Run(context.Background(), root, "dev", Options{})
			if kind == "valid" {
				if result.ExitCode() != 0 || check(t, result, "managed-indices").Status != "passed" {
					t.Fatalf("%+v", result)
				}
			} else if result.ExitCode() == 0 {
				t.Fatalf("invalid managed state passed: %+v", result)
			}
			if !reflect.DeepEqual(before, hashTree(t, root)) {
				t.Fatal("external bytes changed")
			}
		})
	}
}

type deniedReader struct {
	Reader
	deny string
}

func (r deniedReader) ReadFile(name string) ([]byte, error) {
	if name == r.deny {
		return nil, fs.ErrPermission
	}
	return r.Reader.ReadFile(name)
}
func TestPermissionsCancellationBudgetsAndLegacy(t *testing.T) {
	t.Run("permission", func(t *testing.T) {
		root, _ := fixture(t)
		r := Run(context.Background(), root, "dev", Options{Open: func(ctx context.Context, p string, l Limits) (Reader, error) {
			r, e := OpenRoot(ctx, p, l)
			return deniedReader{r, ".archbase/scopes.yaml"}, e
		}})
		if r.ExitCode() != 3 || check(t, r, "workspace-index").Reason != "permission-denied" {
			t.Fatalf("%+v", r)
		}
	})
	t.Run("cancel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		r := Run(ctx, t.TempDir(), "dev", Options{})
		if r.ExitCode() != 130 {
			t.Fatalf("%+v", r)
		}
	})
	t.Run("budget", func(t *testing.T) {
		root, _ := fixture(t)
		r := Run(context.Background(), root, "dev", Options{Limits: Limits{Files: 2, FileBytes: 16, TotalBytes: 32}})
		if r.ExitCode() != 3 {
			t.Fatalf("%+v", r)
		}
	})
	t.Run("legacy", func(t *testing.T) {
		root := tempDir(t)
		put(t, root, "nested/.archbase/scope.yaml", []byte("not loaded"))
		r := Run(context.Background(), root, "dev", Options{})
		if check(t, r, "legacy-indices").Status != "warning" {
			t.Fatalf("%+v", r)
		}
	})
	t.Run("invalid-root", func(t *testing.T) {
		r := Run(context.Background(), filepath.Join(tempDir(t), "missing"), "dev", Options{})
		if r.ExitCode() != 2 || check(t, r, "cli-version").Status != "passed" {
			t.Fatalf("%+v", r)
		}
	})
	t.Run("excluded", func(t *testing.T) {
		root := tempDir(t)
		put(t, root, ".env", []byte("never return"))
		r, err := OpenRoot(context.Background(), root, DefaultLimits)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()
		if _, err = r.ReadFile(".env"); !errors.Is(err, ErrExcluded) {
			t.Fatal(err)
		}
	})
}
