package rules

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRemoveSpecificRuleAcrossFormats(t *testing.T) {
	root := t.TempDir()
	rule := exportTestRule()
	exporter, _ := NewExporter(OSExportFileSystem{})

	for _, relative := range []string{"src/features/AGENTS.md", "src/pages/AGENTS.md"} {
		target := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte("# External instructions\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := exporter.Export(rule, FormatCursor, ExportOptions{Destination: root}); err != nil {
		t.Fatal(err)
	}
	if _, err := exporter.Export(rule, FormatCopilot, ExportOptions{Destination: root}); err != nil {
		t.Fatal(err)
	}
	if _, err := exporter.Export(rule, FormatAgents, ExportOptions{Destination: root, Merge: true}); err != nil {
		t.Fatal(err)
	}

	result, err := exporter.Remove(rule.Definition.ID, RemoveOptions{Destination: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Paths) != 4 || len(result.RuleIDs) != 1 || result.RuleIDs[0] != rule.Definition.ID {
		t.Fatalf("unexpected removal result: %#v", result)
	}
	for _, target := range []string{
		filepath.Join(root, ".cursor", "rules", "architecture-example-1.mdc"),
		filepath.Join(root, ".github", "instructions", "architecture-example-1.instructions.md"),
	} {
		if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("owned export survived removal at %s: %v", target, err)
		}
	}
	for _, relative := range []string{"src/features/AGENTS.md", "src/pages/AGENTS.md"} {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(content, []byte("# External instructions")) || bytes.Contains(content, []byte("archbase:rule")) {
			t.Fatalf("external AGENTS.md content was not preserved: %q", content)
		}
	}

	again, err := exporter.Remove(rule.Definition.ID, RemoveOptions{Destination: root})
	if err != nil || len(again.Paths) != 0 {
		t.Fatalf("idempotent removal: result=%#v err=%v", again, err)
	}
}

func TestRemoveAllManagedRulesPreservesForeignFiles(t *testing.T) {
	root := t.TempDir()
	exporter, _ := NewExporter(OSExportFileSystem{})
	first := exportTestRule()
	second := exportTestRule()
	second.Definition.ID = "architecture/second@1"
	second.Definition.Name = "Second architecture"

	for _, rule := range []Rule{first, second} {
		if _, err := exporter.Export(rule, FormatCursor, ExportOptions{Destination: root}); err != nil {
			t.Fatal(err)
		}
		if _, err := exporter.Export(rule, FormatCopilot, ExportOptions{Destination: root}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := exporter.Export(first, FormatAgents, ExportOptions{Destination: root}); err != nil {
		t.Fatal(err)
	}
	if _, err := exporter.Export(second, FormatAgents, ExportOptions{Destination: root, Merge: true}); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(root, ".cursor", "rules", "team.mdc")
	if err := os.WriteFile(foreign, []byte("team-owned\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := exporter.Remove("", RemoveOptions{Destination: root, All: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.RuleIDs) != 2 || result.RuleIDs[0] != first.Definition.ID || result.RuleIDs[1] != second.Definition.ID {
		t.Fatalf("unexpected removed IDs: %#v", result.RuleIDs)
	}
	if content, err := os.ReadFile(foreign); err != nil || string(content) != "team-owned\n" {
		t.Fatalf("foreign Cursor rule was changed: %q err=%v", content, err)
	}
	for _, relative := range []string{"src/features/AGENTS.md", "src/pages/AGENTS.md"} {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(content), "archbase:rule") {
			t.Fatalf("managed blocks survived in %s: %q", relative, content)
		}
	}
}

func TestRemoveSupportsLegacyExportsWithStrictFingerprint(t *testing.T) {
	root := t.TempDir()
	exporter, _ := NewExporter(OSExportFileSystem{})
	rule := exportTestRule()
	for _, format := range []Format{FormatCursor, FormatCopilot} {
		result, err := exporter.Export(rule, format, ExportOptions{Destination: root})
		if err != nil {
			t.Fatal(err)
		}
		content, err := os.ReadFile(result.Paths[0])
		if err != nil {
			t.Fatal(err)
		}
		legacy := strings.Replace(string(content), ownershipMarker(rule.Definition.ID)+"\n\n", "", 1)
		if err := os.WriteFile(result.Paths[0], []byte(legacy), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	result, err := exporter.Remove(rule.Definition.ID, RemoveOptions{Destination: root})
	if err != nil {
		t.Fatal(err)
	}
	if result.LegacyExports != 2 || len(result.Paths) != 2 || len(result.RuleIDs) != 1 {
		t.Fatalf("unexpected legacy removal: %#v", result)
	}
}

func TestRemoveManagedBlocksPreservesOtherRulesAndRejectsCorruption(t *testing.T) {
	firstStart, firstEnd := managedMarkers("architecture/first@1")
	secondStart, secondEnd := managedMarkers("architecture/second@1")
	content := firstStart + "\nfirst\n" + firstEnd + "\nexternal\n" + secondStart + "\nsecond\n" + secondEnd + "\n"
	updated, ids, changed, err := removeManagedBlocks([]byte(content), "architecture/first@1", false)
	if err != nil || !changed || len(ids) != 1 || ids[0] != "architecture/first@1" {
		t.Fatalf("specific managed removal: updated=%q ids=%v changed=%t err=%v", updated, ids, changed, err)
	}
	if strings.Contains(string(updated), firstStart) || !strings.Contains(string(updated), "external\n") || !strings.Contains(string(updated), secondStart) {
		t.Fatalf("unrelated content was not preserved: %q", updated)
	}

	for _, invalid := range []string{
		firstStart + "\nmissing end\n",
		firstEnd + "\n",
		firstStart + "\n" + secondStart + "\n" + secondEnd + "\n" + firstEnd + "\n",
		firstStart + "\n" + firstEnd + "\n" + firstStart + "\n" + firstEnd + "\n",
	} {
		if _, _, _, err := removeManagedBlocks([]byte(invalid), "", true); err == nil {
			t.Fatalf("corrupt managed blocks were accepted: %q", invalid)
		}
	}
}

func TestRemovePrevalidatesMarkersBeforeChangingFiles(t *testing.T) {
	root := t.TempDir()
	exporter, _ := NewExporter(OSExportFileSystem{})
	rule := exportTestRule()
	result, err := exporter.Export(rule, FormatCursor, ExportOptions{Destination: root})
	if err != nil {
		t.Fatal(err)
	}
	ownedBefore, _ := os.ReadFile(result.Paths[0])
	invalid := filepath.Join(root, "src", "AGENTS.md")
	if err := os.MkdirAll(filepath.Dir(invalid), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(invalid, []byte("<!-- archbase:rule architecture/example@1 start -->\nmissing end\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := exporter.Remove("", RemoveOptions{Destination: root, All: true}); err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("invalid markers were not rejected: %v", err)
	}
	ownedAfter, err := os.ReadFile(result.Paths[0])
	if err != nil || !bytes.Equal(ownedBefore, ownedAfter) {
		t.Fatalf("prevalidation changed an earlier file: err=%v", err)
	}
}

type failingRemoveFS struct {
	OSExportFileSystem
	removes int
	failAt  int
}

func (f *failingRemoveFS) Remove(path string) error {
	f.removes++
	if f.removes == f.failAt {
		return errors.New("injected remove failure")
	}
	return f.OSExportFileSystem.Remove(path)
}

func TestRemoveRollsBackEarlierFiles(t *testing.T) {
	root := t.TempDir()
	rule := exportTestRule()
	writer, _ := NewExporter(OSExportFileSystem{})
	cursor, err := writer.Export(rule, FormatCursor, ExportOptions{Destination: root})
	if err != nil {
		t.Fatal(err)
	}
	copilot, err := writer.Export(rule, FormatCopilot, ExportOptions{Destination: root})
	if err != nil {
		t.Fatal(err)
	}
	remover, _ := NewExporter(&failingRemoveFS{failAt: 2})
	if _, err := remover.Remove(rule.Definition.ID, RemoveOptions{Destination: root}); err == nil {
		t.Fatal("expected injected remove failure")
	}
	for _, target := range []string{cursor.Paths[0], copilot.Paths[0]} {
		if _, err := os.Stat(target); err != nil {
			t.Fatalf("rollback did not restore %s: %v", target, err)
		}
	}
}

func TestRemoveRejectsSymlinkedManagedTarget(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks commonly requires elevated Windows privileges")
	}
	root := t.TempDir()
	target := filepath.Join(root, ".cursor")
	if err := os.Symlink(t.TempDir(), target); err != nil {
		t.Fatal(err)
	}
	exporter, _ := NewExporter(OSExportFileSystem{})
	if _, err := exporter.Remove("", RemoveOptions{Destination: root, All: true}); err == nil {
		t.Fatal("expected symlinked export directory to be rejected")
	}
}
