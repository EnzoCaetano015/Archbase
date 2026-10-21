package rules

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRemoveSpecificAndAllUnifiedFormats(t *testing.T) {
	root := t.TempDir()
	e, _ := NewExporter(OSExportFileSystem{})
	r := exportTestRule()
	other := exportTestRule()
	other.Definition.ID = "architecture/other@1"
	for _, f := range formats {
		install(t, e, root, r, f)
	}
	install(t, e, root, other, FormatClaude)
	put(t, root, ".archbase/patterns/keep.txt", []byte("pattern"))
	put(t, root, ".archbase/rules/foreign.md", []byte("external"))
	put(t, root, ".cursor/rules/team.mdc", []byte("external"))
	scopes := get(t, root, ".archbase/scopes.yaml")
	res, err := e.Remove(r.Definition.ID, RemoveOptions{Destination: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Paths) != 5 || len(res.RuleIDs) != 1 {
		t.Fatalf("%#v", res)
	}
	for _, f := range formats {
		c := string(get(t, root, entryPaths[f]))
		if strings.Contains(c, r.Definition.ID) || !strings.Contains(c, other.Definition.ID) {
			t.Fatal(c)
		}
	}
	res, err = e.Remove("", RemoveOptions{Destination: root, All: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.RuleIDs) != 1 || res.RuleIDs[0] != other.Definition.ID {
		t.Fatalf("%#v", res)
	}
	for _, f := range formats {
		if strings.Contains(string(get(t, root, entryPaths[f])), "archbase:rules") {
			t.Fatal("index survived", f)
		}
	}
	if !bytes.Equal(scopes, get(t, root, ".archbase/scopes.yaml")) || string(get(t, root, ".archbase/patterns/keep.txt")) != "pattern" || string(get(t, root, ".archbase/rules/foreign.md")) != "external" || string(get(t, root, ".cursor/rules/team.mdc")) != "external" {
		t.Fatal("external/workspace changed")
	}
	res, err = e.Remove("", RemoveOptions{Destination: root, All: true})
	if err != nil || len(res.Paths) != 0 {
		t.Fatalf("idempotence: %#v %v", res, err)
	}
}
func TestRemoveMissingDocumentAndPreservesEntryBytes(t *testing.T) {
	root := t.TempDir()
	e, _ := NewExporter(OSExportFileSystem{})
	r := exportTestRule()
	for _, f := range formats {
		content := []byte("external\r\n")
		if f == FormatCursor {
			content = []byte("---\r\ndescription: Team\r\nalwaysApply: true\r\n---\r\n\r\nexternal\r\n")
		}
		put(t, root, entryPaths[f], content)
		install(t, e, root, r, f)
	}
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(documentPath(r.Definition.ID)))); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Remove(r.Definition.ID, RemoveOptions{Destination: root}); err != nil {
		t.Fatal(err)
	}
	for _, f := range formats {
		c := get(t, root, entryPaths[f])
		if !bytes.Contains(c, []byte("external\r\n")) || bytes.Contains(c, []byte("archbase:rules")) {
			t.Fatal(string(c))
		}
	}
}
func TestRemoveLegacyOfflineAndStrictOwnership(t *testing.T) {
	root := t.TempDir()
	e, _ := NewExporter(OSExportFileSystem{})
	r := exportTestRule()
	copilot, err := renderCopilot(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range append(renderCursor(r), copilot...) {
		legacy := strings.Replace(string(a.Content), ownershipMarker(r.Definition.ID)+"\n\n", "", 1)
		put(t, root, a.RelativePath, []byte(legacy))
	}
	agents := renderAgents(r)
	put(t, root, "src/AGENTS.md", agents[0].Content)
	put(t, root, "src/CLAUDE.md", append([]byte("external\r\n"), agents[0].Content...))
	res, err := e.Remove(r.Definition.ID, RemoveOptions{Destination: root})
	if err != nil {
		t.Fatal(err)
	}
	if res.LegacyExports != 2 || len(res.Paths) != 4 {
		t.Fatalf("%#v", res)
	}
	if _, err := os.Stat(filepath.Join(root, "src/AGENTS.md")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}
	if string(get(t, root, "src/CLAUDE.md")) != "external\r\n" {
		t.Fatal("external changed")
	}
	// A marker belonging to a different ID at this ID's filename is not ours to delete.
	a := renderCursor(r)[0]
	a.Content = []byte(ownershipMarker("architecture/foreign@1") + "\nforeign")
	put(t, root, a.RelativePath, a.Content)
	if _, err := e.Remove(r.Definition.ID, RemoveOptions{Destination: root}); err == nil {
		if _, stat := os.Stat(filepath.Join(root, filepath.FromSlash(a.RelativePath))); stat != nil {
			t.Fatal("mismatched file deleted")
		}
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

type failingTransactionFS struct {
	OSExportFileSystem
	operations, failAt int
}

func (f *failingTransactionFS) fail() error {
	f.operations++
	if f.operations == f.failAt {
		return errors.New("injected transaction failure")
	}
	return nil
}
func (f *failingTransactionFS) WriteFileAtomic(p string, d []byte, o bool) error {
	if err := f.fail(); err != nil {
		return err
	}
	return f.OSExportFileSystem.WriteFileAtomic(p, d, o)
}
func (f *failingTransactionFS) Remove(p string) error {
	if filepath.Base(p) != ".archbase.lock" {
		if err := f.fail(); err != nil {
			return err
		}
	}
	return f.OSExportFileSystem.Remove(p)
}
func TestRemoveRollbackEveryStage(t *testing.T) {
	for stage := 1; stage <= 5; stage++ {
		t.Run(fmt.Sprint(stage), func(t *testing.T) {
			root := t.TempDir()
			writer, _ := NewExporter(OSExportFileSystem{})
			for _, f := range formats {
				install(t, writer, root, exportTestRule(), f)
			}
			before := snapshot(t, root)
			e, _ := NewExporter(&failingTransactionFS{failAt: stage})
			if _, err := e.Remove("", RemoveOptions{Destination: root, All: true}); err == nil {
				t.Fatal("expected failure")
			}
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("rollback changed files")
			}
		})
	}
}
func TestRemoveContextAndSymlinks(t *testing.T) {
	root := t.TempDir()
	e, _ := NewExporter(OSExportFileSystem{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := e.RemoveContext(ctx, "", RemoveOptions{Destination: root, All: true}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	target := filepath.Join(root, ".cursor")
	if err := os.Symlink(t.TempDir(), target); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := e.Remove("", RemoveOptions{Destination: root, All: true}); err == nil {
		t.Fatal("symlink accepted")
	}
}

func TestUpdateAndLegacyCleanupRollbackEveryStage(t *testing.T) {
	for stage := 1; stage <= 9; stage++ {
		t.Run(fmt.Sprint(stage), func(t *testing.T) {
			root := t.TempDir()
			r := exportTestRule()
			writer, _ := NewExporter(OSExportFileSystem{})
			for _, f := range formats {
				install(t, writer, root, r, f)
			}
			legacy := renderAgents(r)[0].Content
			put(t, root, "nested/AGENTS.md", legacy)
			put(t, root, "nested/CLAUDE.md", append([]byte("external\r\n"), legacy...))
			copilot, _ := renderCopilot(r)
			for _, a := range append(renderCursor(r), copilot...) {
				put(t, root, a.RelativePath, a.Content)
			}
			before := snapshot(t, root)
			r.Definition.Description = "Updated objective description."
			e, _ := NewExporter(&failingTransactionFS{failAt: stage})
			if _, err := e.Export(r, FormatAgents, ExportOptions{Destination: root, Overwrite: true}); err == nil {
				t.Fatal("expected failure")
			}
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("update/legacy cleanup rollback changed files")
			}
		})
	}
}

type cancelAfterWriteFS struct {
	OSExportFileSystem
	cancel context.CancelFunc
}

func (f *cancelAfterWriteFS) WriteFileAtomic(p string, data []byte, overwrite bool) error {
	err := f.OSExportFileSystem.WriteFileAtomic(p, data, overwrite)
	f.cancel()
	return err
}
func TestCancellationDuringWriteRollsBack(t *testing.T) {
	root := t.TempDir()
	put(t, root, "AGENTS.md", []byte("External instructions\r\n"))
	before := snapshot(t, root)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	e, _ := NewExporter(&cancelAfterWriteFS{cancel: cancel})
	if _, err := e.ExportContext(ctx, exportTestRule(), FormatAgents, ExportOptions{Destination: root}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("cancelled write left changes")
	}
}

type lateDirectoryFailureFS struct{ OSExportFileSystem }

func (f lateDirectoryFailureFS) EnsureDir(p string) error {
	if err := f.OSExportFileSystem.EnsureDir(p); err != nil {
		return err
	}
	if filepath.Base(p) == "rules" {
		return errors.New("late directory creation failure")
	}
	return nil
}
func TestDirectoryCreationFailureRollsBack(t *testing.T) {
	root := t.TempDir()
	e, _ := NewExporter(lateDirectoryFailureFS{})
	if _, err := e.Export(exportTestRule(), FormatAgents, ExportOptions{Destination: root}); err == nil {
		t.Fatal("expected failure")
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("partial workspace: %v %v", entries, err)
	}
}
