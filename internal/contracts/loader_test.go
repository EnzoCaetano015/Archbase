package contracts

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
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

func TestRuleFSAndEncoding(t *testing.T) {
	rule := `schemaVersion: 1
id: architecture/next-modular@1
name: Modular Next
description: Architecture without exporter-specific fields.
version: 1.0.0
scopes:
  - path: src/pages/**
    pattern: next/page@1234
rules:
  - Pages must have their own directory.
metadata:
  owner: platform
`
	loaded, err := LoadRuleFS(fstest.MapFS{"rule.yaml": &fstest.MapFile{Data: []byte(rule)}}, "rule.yaml")
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeRule(loaded)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) == 0 || loaded.Metadata["owner"] != "platform" {
		t.Fatalf("unexpected rule round trip: %#v", loaded)
	}
}

func TestRuleSemanticValidation(t *testing.T) {
	valid := `schemaVersion: 1
id: architecture/example@1
name: Example
version: 1.0.0
scopes:
  - path: src/pages/**
    pattern: next/page@1234
rules:
  - Keep architecture explicit.
`
	tests := []struct {
		name string
		from string
		to   string
		want string
	}{
		{name: "absolute path", from: "src/pages/**", to: "/src/pages/**", want: "/scopes/0/path"},
		{name: "traversal", from: "src/pages/**", to: "../pages/**", want: "/scopes/0/path"},
		{name: "backslash", from: "src/pages/**", to: `src\pages\**`, want: "/scopes/0/path"},
		{name: "not normalized", from: "src/pages/**", to: "src/./pages/**", want: "/scopes/0/path"},
		{name: "invalid glob", from: "src/pages/**", to: "src/[pages/**", want: "/scopes/0/path"},
		{name: "whitespace restriction", from: "Keep architecture explicit.", to: "   ", want: "/rules/0"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := LoadRule(writeDocument(t, "rule.yaml", strings.Replace(valid, test.from, test.to, 1)))
			assertValidationError(t, err, test.want)
		})
	}
	duplicate := strings.Replace(valid, "rules:", "  - path: src/pages/**\n    pattern: next/page@1234\nrules:", 1)
	_, err := LoadRule(writeDocument(t, "rule.yaml", duplicate))
	assertValidationError(t, err, "/scopes/1")
}

func TestRuleSchemaValidation(t *testing.T) {
	valid := `schemaVersion: 1
id: architecture/example@1
name: Example
version: 1.0.0
scopes:
  - path: src/**
    pattern: next/page@1234
rules:
  - Keep architecture explicit.
`
	tests := []struct {
		name string
		edit func(string) string
		want string
	}{
		{name: "missing name", edit: func(value string) string { return strings.Replace(value, "name: Example\n", "", 1) }, want: "name"},
		{name: "invalid ID", edit: func(value string) string {
			return strings.Replace(value, "architecture/example@1", "Architecture/example@1", 1)
		}, want: "id"},
		{name: "invalid SemVer", edit: func(value string) string { return strings.Replace(value, "version: 1.0.0", "version: one", 1) }, want: "version"},
		{name: "unknown field", edit: func(value string) string { return value + "exporter: cursor\n" }, want: "exporter"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := LoadRule(writeDocument(t, "rule.yaml", test.edit(valid)))
			assertValidationError(t, err, test.want)
		})
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
