package rules

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/EnzoCaetano015/Archbase/internal/contracts"
)

func exportTestRule() Rule {
	return Rule{Source: "official-embedded", Definition: contracts.Rule{
		SchemaVersion: 1, ID: "architecture/example@1", Name: "Example architecture",
		Description: "Keeps layers separate.", Version: "1.0.0",
		Scopes: []contracts.RuleScope{
			{Path: "src/features/**", Pattern: "next/component@4821"},
			{Path: "src/features/*.test.ts", Pattern: "next/util@3378"},
			{Path: "src/pages/**", Pattern: "next/page@1234"},
		},
		Rules: []string{"Pages depend on features, never the reverse."},
	}}
}

func TestRenderCursorAndCopilot(t *testing.T) {
	rule := exportTestRule()
	for _, test := range []struct {
		format Format
		path   string
		want   []string
	}{
		{FormatCursor, ".cursor/rules/architecture-example-1.mdc", []string{"description: \"Keeps layers separate.\"", "globs:", "alwaysApply: false"}},
		{FormatCopilot, ".github/instructions/architecture-example-1.instructions.md", []string{"applyTo: \"src/features/**,src/features/*.test.ts,src/pages/**\""}},
	} {
		artifacts, err := Render(rule, test.format)
		if err != nil || len(artifacts) != 1 {
			t.Fatalf("render %s: artifacts=%d err=%v", test.format, len(artifacts), err)
		}
		if artifacts[0].RelativePath != test.path {
			t.Errorf("render %s path = %q", test.format, artifacts[0].RelativePath)
		}
		text := string(artifacts[0].Content)
		for _, expected := range append(test.want, "`next/component@4821`", "`arc resolve <target-path>`", "Pages depend on features") {
			if !strings.Contains(text, expected) {
				t.Errorf("render %s missing %q:\n%s", test.format, expected, text)
			}
		}
		if strings.Contains(text, "export default") || strings.Contains(text, "namespace Archbase") {
			t.Errorf("render %s copied pattern example code", test.format)
		}
	}
}

func TestRenderAgentsGroupsStaticScopePrefixes(t *testing.T) {
	artifacts, err := Render(exportTestRule(), FormatAgents)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"src/features/AGENTS.md", "src/pages/AGENTS.md"}
	if len(artifacts) != len(want) {
		t.Fatalf("got %d artifacts, want %d", len(artifacts), len(want))
	}
	for index := range want {
		if artifacts[index].RelativePath != want[index] {
			t.Errorf("artifact %d path=%q want=%q", index, artifacts[index].RelativePath, want[index])
		}
		if !bytes.Contains(artifacts[index].Content, []byte("<!-- archbase:rule architecture/example@1 start -->")) {
			t.Errorf("artifact %s has no managed marker", artifacts[index].RelativePath)
		}
	}
	if strings.Count(string(artifacts[0].Content), "src/features/") != 2 {
		t.Errorf("same-directory scopes were not grouped:\n%s", artifacts[0].Content)
	}
}

func TestExporterConflictsOverwriteAndCreatesDestination(t *testing.T) {
	root := filepath.Join(t.TempDir(), "new-project")
	exporter, _ := NewExporter(OSExportFileSystem{})
	result, err := exporter.Export(exportTestRule(), FormatCursor, ExportOptions{Destination: root})
	if err != nil || len(result.Paths) != 1 {
		t.Fatalf("first export: result=%#v err=%v", result, err)
	}
	if _, err := os.Stat(result.Paths[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := exporter.Export(exportTestRule(), FormatCursor, ExportOptions{Destination: root}); err == nil {
		t.Fatal("expected existing Cursor export to conflict")
	}
	updated := exportTestRule()
	updated.Definition.Rules = []string{"Updated restriction."}
	if _, err := exporter.Export(updated, FormatCursor, ExportOptions{Destination: root, Overwrite: true}); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(result.Paths[0])
	if !bytes.Contains(content, []byte("Updated restriction.")) {
		t.Fatal("overwrite did not replace content")
	}
}

func TestAgentsMergePreservesContentAndIsIdempotent(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "src", "pages", "AGENTS.md")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("# Team instructions\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	exporter, _ := NewExporter(OSExportFileSystem{})
	if _, err := exporter.Export(exportTestRule(), FormatAgents, ExportOptions{Destination: root}); err == nil {
		t.Fatal("expected existing AGENTS.md to require --merge")
	}
	options := ExportOptions{Destination: root, Merge: true}
	if _, err := exporter.Export(exportTestRule(), FormatAgents, options); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(target)
	if !bytes.Contains(first, []byte("# Team instructions")) {
		t.Fatal("external content was not preserved")
	}
	if _, err := exporter.Export(exportTestRule(), FormatAgents, options); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(target)
	if !bytes.Equal(first, second) {
		t.Fatal("managed merge is not idempotent")
	}
}

func TestAgentsMergeRejectsInvalidMarkersAndOverwrite(t *testing.T) {
	artifact := Artifact{Content: []byte("start\nbody\nend\n"), StartMarker: "start", EndMarker: "end"}
	for _, existing := range []string{"start\n", "end\n", "start\nstart\nend\n", "end\nstart\n"} {
		if _, err := mergeManagedBlock([]byte(existing), artifact); err == nil {
			t.Errorf("expected invalid markers in %q", existing)
		}
	}
	exporter, _ := NewExporter(OSExportFileSystem{})
	if _, err := exporter.Export(exportTestRule(), FormatAgents, ExportOptions{Destination: t.TempDir(), Overwrite: true}); err == nil {
		t.Fatal("expected --overwrite to be rejected for agents")
	}
	if _, err := exporter.Export(exportTestRule(), FormatCursor, ExportOptions{Destination: t.TempDir(), Merge: true}); err == nil {
		t.Fatal("expected --merge to be rejected for cursor")
	}
}

func TestExporterRejectsTraversalAndSymlinks(t *testing.T) {
	exporter, _ := NewExporter(OSExportFileSystem{})
	root := t.TempDir()
	for _, invalid := range []string{"../outside", "safe/../outside", "/absolute", "C:/absolute", `safe\outside`} {
		if _, err := exporter.writeArtifacts(root, []Artifact{{RelativePath: invalid, Content: []byte("bad")}}, FormatCursor, ExportOptions{}); err == nil {
			t.Errorf("expected %q to be rejected", invalid)
		}
	}
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks commonly requires elevated Windows privileges")
	}
	linked := filepath.Join(root, ".cursor")
	if err := os.Symlink(t.TempDir(), linked); err != nil {
		t.Fatal(err)
	}
	if _, err := exporter.Export(exportTestRule(), FormatCursor, ExportOptions{Destination: root}); err == nil {
		t.Fatal("expected symlink to be rejected")
	}
}

type failingExportFS struct {
	OSExportFileSystem
	writes int
	failAt int
}

func (f *failingExportFS) WriteFileAtomic(path string, data []byte, overwrite bool) error {
	f.writes++
	if f.writes == f.failAt {
		return errors.New("injected write failure")
	}
	return f.OSExportFileSystem.WriteFileAtomic(path, data, overwrite)
}

func TestExporterRollsBackMultipleFiles(t *testing.T) {
	root := t.TempDir()
	fsys := &failingExportFS{failAt: 2}
	exporter, _ := NewExporter(fsys)
	if _, err := exporter.Export(exportTestRule(), FormatAgents, ExportOptions{Destination: root}); err == nil {
		t.Fatal("expected injected write failure")
	}
	if _, err := os.Stat(filepath.Join(root, "src", "features", "AGENTS.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("first file survived rollback: %v", err)
	}
}

func TestExporterRestoresExistingFilesOnRollback(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "src", "features", "AGENTS.md")
	second := filepath.Join(root, "src", "pages", "AGENTS.md")
	for _, target := range []string{first, second} {
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte("external\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	fsys := &failingExportFS{failAt: 2}
	exporter, _ := NewExporter(fsys)
	if _, err := exporter.Export(exportTestRule(), FormatAgents, ExportOptions{Destination: root, Merge: true}); err == nil {
		t.Fatal("expected injected write failure")
	}
	content, err := os.ReadFile(first)
	if err != nil || string(content) != "external\n" {
		t.Fatalf("existing file was not restored: %q err=%v", content, err)
	}
	content, err = os.ReadFile(second)
	if err != nil || string(content) != "external\n" {
		t.Fatalf("failing destination was not restored: %q err=%v", content, err)
	}
}
