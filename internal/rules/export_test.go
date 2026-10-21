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
	"sync"
	"testing"
	"time"

	"github.com/EnzoCaetano015/Archbase/internal/contracts"
	"github.com/EnzoCaetano015/Archbase/internal/registry"
	"github.com/EnzoCaetano015/Archbase/internal/workspace"
)

func exportTestRule() Rule {
	return Rule{Source: "official-embedded", Definition: contracts.Rule{SchemaVersion: 1, ID: "architecture/example@1", Name: "Example architecture", Description: "Keeps layers separate.", Version: "1.0.0",
		Scopes: []contracts.RuleScope{{Path: "src/features/**", Pattern: "next/component@4821"}, {Path: "src/features/*.test.ts", Pattern: "next/util@3378"}, {Path: "src/pages/**", Pattern: "next/page@1234"}},
		Rules:  []string{"Pages depend on features, never the reverse."}}}
}
func put(t *testing.T, root, relative string, content []byte) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, content, 0644); err != nil {
		t.Fatal(err)
	}
}
func get(t *testing.T, root, relative string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	result := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		relative, _ := filepath.Rel(root, p)
		data, err := os.ReadFile(p)
		result[relative] = string(data)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}
func install(t *testing.T, e *Exporter, root string, r Rule, f Format) ExportResult {
	t.Helper()
	res, err := e.Export(r, f, ExportOptions{Destination: root})
	if err != nil {
		t.Fatal(err)
	}
	return res
}
func TestRenderUnifiedFormats(t *testing.T) {
	for _, f := range formats {
		t.Run(string(f), func(t *testing.T) {
			a, err := Render(exportTestRule(), f)
			if err != nil {
				t.Fatal(err)
			}
			if len(a) != 2 || a[0].RelativePath != documentPath(exportTestRule().Definition.ID) || a[1].RelativePath != entryPaths[f] {
				t.Fatalf("%#v", a)
			}
			body, index := string(a[0].Content), string(a[1].Content)
			for _, s := range []string{"id: architecture/example@1", "source: official-embedded", "scopes:", "Pages depend on features", "next/component@4821", "arc resolve", "arc inspect", ownershipMarker("architecture/example@1")} {
				if !strings.Contains(body, s) {
					t.Fatalf("missing %q: %s", s, body)
				}
			}
			prefix := ""
			if f == FormatCopilot {
				prefix = "../"
			}
			if f == FormatCursor {
				prefix = "../../"
			}
			if !strings.Contains(index, "("+prefix+a[0].RelativePath+")") || !strings.Contains(index, "Before changing files, open") {
				t.Fatal(index)
			}
			if strings.Contains(index, "Pages depend on features") || strings.Contains(body, "export default") {
				t.Fatal("copied body/examples")
			}
			if f == FormatCursor && !strings.Contains(index, "alwaysApply: true") {
				t.Fatal(index)
			}
		})
	}
}
func TestActivationSynchronizationIdempotenceAndPreservation(t *testing.T) {
	root := t.TempDir()
	e, _ := NewExporter(OSExportFileSystem{})
	r := exportTestRule()
	original := []byte("# Framework\r\nPreserve bytes.\r\n")
	put(t, root, "AGENTS.md", original)
	put(t, root, "CLAUDE.md", []byte("Not active\r\n"))
	install(t, e, root, r, FormatAgents)
	if string(get(t, root, "CLAUDE.md")) != "Not active\r\n" {
		t.Fatal("activated external file")
	}
	for _, f := range formats {
		install(t, e, root, r, f)
	}
	before := snapshot(t, root)
	again := install(t, e, root, r, FormatClaude)
	if len(again.Paths) != 0 || !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("not idempotent")
	}
	second := exportTestRule()
	second.Definition.ID = "architecture/another@1"
	second.Definition.Description = ""
	install(t, e, root, second, FormatCopilot)
	for _, f := range formats {
		c := string(get(t, root, entryPaths[f]))
		if strings.Index(c, second.Definition.ID) > strings.Index(c, r.Definition.ID) || !strings.Contains(c, second.Definition.Name) {
			t.Fatal(c)
		}
	}
	entries, err := os.ReadDir(filepath.Join(root, ".archbase", "rules"))
	if err != nil || len(entries) != 2 {
		t.Fatalf("%v %v", entries, err)
	}
	if !bytes.HasPrefix(get(t, root, "AGENTS.md"), original) {
		t.Fatal("CRLF changed")
	}
	suffix := []byte("\r\n# User suffix\r\n")
	put(t, root, "AGENTS.md", append(get(t, root, "AGENTS.md"), suffix...))
	install(t, e, root, r, FormatAgents)
	if !bytes.HasSuffix(get(t, root, "AGENTS.md"), suffix) {
		t.Fatal("suffix lost")
	}
}
func TestConflictsOverwriteAndOwnership(t *testing.T) {
	for _, f := range formats {
		t.Run(string(f), func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "new")
			e, _ := NewExporter(OSExportFileSystem{})
			r := exportTestRule()
			install(t, e, root, r, f)
			r.Definition.Rules = []string{"Changed restriction."}
			if _, err := e.Export(r, f, ExportOptions{Destination: root}); err == nil {
				t.Fatal("expected conflict")
			}
			if _, err := e.Export(r, f, ExportOptions{Destination: root, Overwrite: true, Merge: true}); err != nil {
				t.Fatal(err)
			}
			put(t, root, documentPath(r.Definition.ID), []byte("user file"))
			if _, err := e.Export(r, f, ExportOptions{Destination: root, Overwrite: true}); err == nil {
				t.Fatal("replaced foreign document")
			}
		})
	}
	root := t.TempDir()
	e, _ := NewExporter(OSExportFileSystem{})
	a := exportTestRule()
	a.Definition.ID = "a-b/c@1"
	install(t, e, root, a, FormatAgents)
	b := exportTestRule()
	b.Definition.ID = "a/b-c@1"
	if _, err := e.Export(b, FormatAgents, ExportOptions{Destination: root, Overwrite: true}); err == nil {
		t.Fatal("slug collision")
	}
}
func TestCursorFrontmatterAndEscaping(t *testing.T) {
	for _, h := range []string{"external\n", "---\ndescription: user\nalwaysApply: false\n---\n", "---\nalwaysApply: true\nalwaysApply: true\n---\n"} {
		root := t.TempDir()
		put(t, root, entryPaths[FormatCursor], []byte(h))
		before := snapshot(t, root)
		e, _ := NewExporter(OSExportFileSystem{})
		if _, err := e.Export(exportTestRule(), FormatCursor, ExportOptions{Destination: root, Overwrite: true}); err == nil {
			t.Fatal("invalid header accepted")
		}
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("conflict changed files")
		}
	}
	root := t.TempDir()
	h := "---\r\ndescription: Team\r\nalwaysApply: true\r\ncustom: retained\r\n---\r\n\r\nExternal\r\n"
	put(t, root, entryPaths[FormatCursor], []byte(h))
	e, _ := NewExporter(OSExportFileSystem{})
	r := exportTestRule()
	r.Definition.Description = "[unsafe](url) <tag> *bold*\nnext"
	install(t, e, root, r, FormatCursor)
	c := get(t, root, entryPaths[FormatCursor])
	if !bytes.HasPrefix(c, []byte(h)) || !strings.Contains(string(c), "\\[unsafe\\]") || !strings.Contains(string(c), "&lt;tag&gt;") {
		t.Fatal(string(c))
	}
	// A compatible frontmatter-only file need not have a trailing newline.
	root = t.TempDir()
	h = "---\nalwaysApply: true\n---"
	put(t, root, entryPaths[FormatCursor], []byte(h))
	install(t, e, root, exportTestRule(), FormatCursor)
	if !bytes.HasPrefix(get(t, root, entryPaths[FormatCursor]), []byte(h)) {
		t.Fatal("frontmatter bytes changed")
	}
}
func TestLegacyCleanupSelectedRuleAndProject(t *testing.T) {
	root := t.TempDir()
	r := exportTestRule()
	other := exportTestRule()
	other.Definition.ID = "architecture/other@1"
	a := renderAgents(r)
	b := renderAgents(other)
	put(t, root, "src/features/AGENTS.md", append([]byte("External\r\n"), append(a[0].Content, b[0].Content...)...))
	put(t, root, "src/pages/CLAUDE.md", a[0].Content)
	put(t, root, "AGENTS.md", a[0].Content)
	cursor := renderCursor(r)
	copilot, err := renderCopilot(r)
	if err != nil {
		t.Fatal(err)
	}
	put(t, root, cursor[0].RelativePath, cursor[0].Content)
	put(t, root, copilot[0].RelativePath, copilot[0].Content)
	put(t, root, "nested/.git", []byte("gitdir: elsewhere"))
	put(t, root, "nested/AGENTS.md", a[0].Content)
	put(t, root, "node_modules/pkg/AGENTS.md", a[0].Content)
	e, _ := NewExporter(OSExportFileSystem{})
	install(t, e, root, r, FormatAgents)
	if _, err := os.Stat(filepath.Join(root, "src/pages/CLAUDE.md")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("legacy-only survived", err)
	}
	mixed := string(get(t, root, "src/features/AGENTS.md"))
	if !strings.HasPrefix(mixed, "External\r\n") || strings.Contains(mixed, r.Definition.ID) || !strings.Contains(mixed, other.Definition.ID) {
		t.Fatal(mixed)
	}
	for _, p := range []string{cursor[0].RelativePath, copilot[0].RelativePath} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(p))); !errors.Is(err, fs.ErrNotExist) {
			t.Fatal("legacy survived", p)
		}
	}
	if !bytes.Equal(get(t, root, "nested/AGENTS.md"), a[0].Content) || !bytes.Equal(get(t, root, "node_modules/pkg/AGENTS.md"), a[0].Content) {
		t.Fatal("crossed boundary")
	}
}
func TestMarkersPathsAndSymlinksPrevalidated(t *testing.T) {
	for _, s := range []string{indexStart + "\n", indexEnd + "\n", indexEnd + "\n" + indexStart + "\n", indexStart + "\n" + indexEnd + "\n" + indexStart + "\n" + indexEnd + "\n", indexStart + "\n<!-- archbase:rule architecture/example@1 start -->\n" + indexEnd + "\n"} {
		root := t.TempDir()
		put(t, root, "CLAUDE.md", []byte(s))
		e, _ := NewExporter(OSExportFileSystem{})
		before := snapshot(t, root)
		if _, err := e.Export(exportTestRule(), FormatAgents, ExportOptions{Destination: root}); err == nil {
			t.Fatal("markers accepted", s)
		}
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("prevalidation wrote files")
		}
	}
	for _, p := range []string{"../outside", "safe/../outside", "/absolute", "C:/absolute", `safe\outside`} {
		if _, err := confinedTarget(t.TempDir(), p); err == nil {
			t.Fatal("unsafe", p)
		}
	}
	for _, relative := range []string{".archbase", ".cursor", ".github", "AGENTS.md"} {
		root := t.TempDir()
		target := t.TempDir()
		if relative == "AGENTS.md" {
			put(t, target, "file", []byte("external"))
			target = filepath.Join(target, "file")
		}
		if err := os.Symlink(target, filepath.Join(root, relative)); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		e, _ := NewExporter(OSExportFileSystem{})
		if _, err := e.Export(exportTestRule(), FormatAgents, ExportOptions{Destination: root}); err == nil {
			t.Fatal("symlink", relative)
		}
	}
}

type failingExportFS struct {
	OSExportFileSystem
	writes, failAt int
}

func (f *failingExportFS) WriteFileAtomic(p string, d []byte, o bool) error {
	f.writes++
	if f.writes == f.failAt {
		return errors.New("injected write failure")
	}
	return f.OSExportFileSystem.WriteFileAtomic(p, d, o)
}
func TestExportRollbackEveryWrite(t *testing.T) {
	for stage := 1; stage <= 3; stage++ {
		t.Run(fmt.Sprint(stage), func(t *testing.T) {
			root := t.TempDir()
			put(t, root, "AGENTS.md", []byte("external\r\n"))
			before := snapshot(t, root)
			e, _ := NewExporter(&failingExportFS{failAt: stage})
			if _, err := e.Export(exportTestRule(), FormatAgents, ExportOptions{Destination: root}); err == nil {
				t.Fatal("expected failure")
			}
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("rollback changed files")
			}
			if _, err := os.Stat(filepath.Join(root, ".archbase")); !errors.Is(err, fs.ErrNotExist) {
				t.Fatal("directories remain", err)
			}
		})
	}
}
func TestContextAndConcurrentInstallation(t *testing.T) {
	root := t.TempDir()
	e, _ := NewExporter(OSExportFileSystem{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := e.ExportContext(ctx, exportTestRule(), FormatAgents, ExportOptions{Destination: root}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	put(t, root, ".archbase.lock", nil)
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if _, err := e.ExportContext(ctx, exportTestRule(), FormatAgents, ExportOptions{Destination: root}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, ".archbase.lock")); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	failures := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r := exportTestRule()
			r.Definition.ID = fmt.Sprintf("architecture/rule-%d@1", i)
			_, err := e.Export(r, formats[i%len(formats)], ExportOptions{Destination: root})
			failures <- err
		}(i)
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range formats {
		if c := string(get(t, root, entryPaths[f])); strings.Count(c, "[rule](") != 8 {
			t.Fatal(c)
		}
	}
	if _, err := os.Stat(filepath.Join(root, ".archbase.lock")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("lock remained", err)
	}
}

func TestRulesFirstThenPatternsAndSharedConcurrentLock(t *testing.T) {
	root := t.TempDir()
	e, _ := NewExporter(OSExportFileSystem{})
	install(t, e, root, exportTestRule(), FormatAgents)
	source, err := registry.NewEmbeddedSource()
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := registry.NewResolver(source)
	if err != nil {
		t.Fatal(err)
	}
	service, err := workspace.NewService(workspace.OSFileSystem{}, resolver)
	if err != nil {
		t.Fatal(err)
	}
	service = service.WithProjectRoot(root)
	if _, err = service.Add(context.Background(), "next/page@1234", root); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	failures := make(chan error, 8)
	for i := 0; i < 4; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			_, err := service.Create(context.Background(), fmt.Sprintf("custom-%d", i), filepath.Join(root, fmt.Sprintf("scope-%d", i)), "")
			failures <- err
		}(i)
		go func(i int) {
			defer wg.Done()
			r := exportTestRule()
			r.Definition.ID = fmt.Sprintf("architecture/concurrent-%d@1", i)
			_, err := e.Export(r, FormatClaude, ExportOptions{Destination: root})
			failures <- err
		}(i)
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	index, err := contracts.LoadWorkspaceFS(os.DirFS(filepath.Join(root, ".archbase")), "scopes.yaml")
	if err != nil || len(index.Scopes) != 5 {
		t.Fatalf("lost scopes: %#v %v", index, err)
	}
	if strings.Count(string(get(t, root, "AGENTS.md")), "[rule](") != 5 {
		t.Fatal("lost rules")
	}
}

func TestCorruptCentralDocumentsPrevalidated(t *testing.T) {
	r := exportTestRule()
	a, err := renderDocument(r)
	if err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"metadata-id":       strings.Replace(string(a.Content), "id: architecture/example@1", "id: architecture/other@1", 1),
		"version":           strings.Replace(string(a.Content), "version: 1.0.0", "version: nope", 1),
		"scope":             strings.Replace(string(a.Content), "src/features/**", "../escape", 1),
		"duplicate-marker":  string(a.Content) + "\n" + ownershipMarker(r.Definition.ID) + "\n",
		"incomplete-marker": string(a.Content) + "\n<!-- archbase:rule invalid\n",
		"index-marker":      string(a.Content) + "\n" + indexStart + "\n",
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			put(t, root, a.RelativePath, []byte(content))
			before := snapshot(t, root)
			e, _ := NewExporter(OSExportFileSystem{})
			if _, err := e.Export(r, FormatAgents, ExportOptions{Destination: root, Overwrite: true}); err == nil {
				t.Fatal("corruption accepted")
			}
			if _, err := e.Remove("", RemoveOptions{Destination: root, All: true}); err == nil {
				t.Fatal("corruption deleted")
			}
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("corruption changed files")
			}
		})
	}
}
