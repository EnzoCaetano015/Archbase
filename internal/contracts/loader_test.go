package contracts

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validManifest = `schemaVersion: 1
id: next/hook@1000
name: Hook
type: pattern
version: 1.0.0
structure:
  root: "{{Name}}"
  files:
    - source: Example.ts
      destination: "{{Name}}.ts"
      required: true
allowedChanges: [identifiers]
preserve: [declaration-order]
metadata:
  extension: accepted
`

func writeDocument(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadManifestAcceptsSingleFilePatternAndMetadata(t *testing.T) {
	manifest, err := LoadManifest(writeDocument(t, "manifest.yaml", validManifest))
	if err != nil {
		t.Fatal(err)
	}
	if manifest.ID != "next/hook@1000" || manifest.Metadata["extension"] != "accepted" {
		t.Fatalf("unexpected manifest: %#v", manifest)
	}
}

func TestLoadManifestAcceptsBundle(t *testing.T) {
	bundle := strings.Replace(validManifest, "type: pattern", "type: pattern-bundle", 1)
	bundle = strings.Replace(bundle, "allowedChanges:", "    - source: Example.utils.ts\n      destination: \"{{Name}}.utils.ts\"\n      required: true\nallowedChanges:", 1)
	if _, err := LoadManifest(writeDocument(t, "manifest.yaml", bundle)); err != nil {
		t.Fatal(err)
	}
}

func TestLoadManifestRejectsUnknownCoreField(t *testing.T) {
	invalid := validManifest + "unexpected: true\n"
	_, err := LoadManifest(writeDocument(t, "manifest.yaml", invalid))
	assertValidationError(t, err, "unexpected")
}

func TestLoadManifestRejectsInvalidIDVersionAndEscapingPath(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"id", strings.Replace(validManifest, "next/hook@1000", "Next/Hook@1000", 1), "id"},
		{"version", strings.Replace(validManifest, "1.0.0", "v1", 1), "version"},
		{"path", strings.Replace(validManifest, "Example.ts", "../Example.ts", 1), "source"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := LoadManifest(writeDocument(t, "manifest.yaml", test.content))
			assertValidationError(t, err, test.want)
		})
	}
}

func TestLoadScopeAndRule(t *testing.T) {
	scope := `schemaVersion: 1
scope:
  path: "./**/*"
pattern:
  id: next/page@1234
  source: local
  root: ./patterns/page-1234
behavior:
  nearestScopeWins: true
  allowLocalCustomization: true
`
	if _, err := LoadScope(writeDocument(t, "scope.yaml", scope)); err != nil {
		t.Fatal(err)
	}
	rule := `schemaVersion: 1
id: architecture/next-modular@1
name: Modular Next
version: 1.0.0
scopes:
  - path: src/pages/**
    pattern: next/page@1234
rules:
  - Pages must have their own directory.
`
	if _, err := LoadRule(writeDocument(t, "rule.yaml", rule)); err != nil {
		t.Fatal(err)
	}
}

func TestLocalScopeRequiresRoot(t *testing.T) {
	invalid := `schemaVersion: 1
scope:
  path: "./**/*"
pattern:
  id: next/page@1234
  source: local
behavior:
  nearestScopeWins: true
  allowLocalCustomization: true
`
	_, err := LoadScope(writeDocument(t, "scope.yaml", invalid))
	assertValidationError(t, err, "root")
}

func assertValidationError(t *testing.T, err error, expected string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected validation error containing %q", expected)
	}
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("error %q does not contain %q", err, expected)
	}
}
