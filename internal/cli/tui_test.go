package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/EnzoCaetano015/Archbase/internal/contracts"
	archrules "github.com/EnzoCaetano015/Archbase/internal/rules"
	"github.com/charmbracelet/x/ansi"
)

func testUI(t *testing.T) *uiModel {
	t.Helper()
	return newUI(context.Background(), operations{defaultDependencies(), RegistryOptions{Ref: "main", CacheRoot: defaultCacheRoot()}}, "test", t.TempDir(), false)
}

func keyPress(code rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code} }
func ctrl(code rune) tea.KeyPressMsg     { return tea.KeyPressMsg{Code: code, Mod: tea.ModCtrl} }

func finishOperation(t *testing.T, m *uiModel, cmd tea.Cmd) operationMsg {
	t.Helper()
	batch, ok := cmd().(tea.BatchMsg)
	if !ok || len(batch) != 2 {
		t.Fatal("expected an asynchronous operation and spinner")
	}
	msg, ok := batch[1]().(operationMsg)
	if !ok {
		t.Fatal("missing typed operation result")
	}
	m.Update(msg)
	return msg
}

func setField(t *testing.T, m *uiModel, id, value string) {
	t.Helper()
	for i := range m.fields {
		f := &m.fields[i]
		if f.id != id {
			continue
		}
		if f.checkbox {
			f.checked = value == "true"
		} else if len(f.choices) > 0 {
			for j, choice := range f.choices {
				if choice == value {
					f.choice = j
					return
				}
			}
			t.Fatalf("unknown choice %q", value)
		} else {
			f.input.SetValue(value)
		}
		return
	}
	t.Fatalf("field %q not found", id)
}

func TestUICatalogToInstallAndResolve(t *testing.T) {
	m := testUI(t)
	_, cmd := m.activate("patterns")
	msg := finishOperation(t, m, cmd)
	if msg.err != nil || len(m.entries) == 0 {
		t.Fatalf("catalog: %v", msg.err)
	}
	// Search by the manifest name, which may differ from its canonical ID.
	m.query.SetValue(m.entries[0].name)
	if len(m.filteredEntries()) == 0 {
		t.Fatal("name search has no matches")
	}
	m.query.SetValue("next/page@1234")
	_, cmd = m.Update(keyPress(tea.KeyEnter))
	msg = finishOperation(t, m, cmd)
	if msg.err != nil || m.screen != detailScreen {
		t.Fatalf("inspect: %v", msg.err)
	}
	m.Update(keyPress('a'))
	if m.formValue("id") != "next/page@1234" {
		t.Fatal("selection was lost")
	}
	m.Update(ctrl('s'))
	if m.screen != confirmScreen || m.busy {
		t.Fatal("install must wait for confirmation")
	}
	if _, err := os.Stat(filepath.Join(m.project, ".archbase")); !os.IsNotExist(err) {
		t.Fatal("review wrote files")
	}
	_, cmd = m.Update(keyPress(tea.KeyEnter))
	generation := m.generation
	_, duplicate := m.Update(keyPress(tea.KeyEnter))
	if duplicate != nil || m.generation != generation {
		t.Fatal("repeated Enter submitted a duplicate write")
	}
	msg = finishOperation(t, m, cmd)
	if msg.err != nil || m.result.Installed == nil {
		t.Fatalf("install: %v", msg.err)
	}
	m.activate("resolve")
	setField(t, m, "path", "src/pages/Home.tsx")
	_, cmd = m.reviewForm()
	msg = finishOperation(t, m, cmd)
	if msg.err != nil || m.result.Resolution.Pattern.Entry.ID.String() != "next/page@1234" {
		t.Fatalf("resolve: %v", msg.err)
	}
}

func TestUIExportAllFormatsAndRemovePreservesExternalContent(t *testing.T) {
	m := testUI(t)
	agents := filepath.Join(m.project, "AGENTS.md")
	if err := os.WriteFile(agents, []byte("# My instructions\r\nPreserve this content.\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, format := range []string{"agents", "claude", "cursor", "copilot"} {
		m.openForm("rules add", "architecture/next-modular@1")
		setField(t, m, "format", format)
		if m.formValue("overwrite") != "false" {
			t.Fatal("overwrite must be opt-in")
		}
		m.reviewForm()
		_, cmd := m.Update(keyPress(tea.KeyEnter))
		if msg := finishOperation(t, m, cmd); msg.err != nil {
			t.Fatal(msg.err)
		}
	}
	m.openForm("remove", "architecture/next-modular@1")
	m.reviewForm()
	_, cmd := m.Update(keyPress(tea.KeyEnter))
	if msg := finishOperation(t, m, cmd); msg.err != nil {
		t.Fatal(msg.err)
	}
	data, err := os.ReadFile(agents)
	if err != nil || !bytes.Contains(data, []byte("Preserve this content.\r\n")) || bytes.Contains(data, []byte("archbase:rule")) {
		t.Fatalf("external changed: %q %v", data, err)
	}
}

func TestUICreateAndCollisionRecovery(t *testing.T) {
	m := testUI(t)
	for attempt := range 2 {
		m.openForm("create", "")
		setField(t, m, "id", "my-page")
		setField(t, m, "from", "next/page@1234")
		m.reviewForm()
		_, cmd := m.Update(keyPress(tea.KeyEnter))
		msg := finishOperation(t, m, cmd)
		if attempt == 0 && msg.err != nil {
			t.Fatal(msg.err)
		}
		if attempt == 1 && (msg.err == nil || m.screen != messageScreen) {
			t.Fatal("collision must be recoverable")
		}
	}
	m.Update(keyPress(tea.KeyEscape))
	if m.formValue("id") != "my-page" {
		t.Fatal("collision discarded the user's input")
	}
}

func TestUIReadCancellationAndStaleResults(t *testing.T) {
	m := testUI(t)
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel, m.busy, m.generation = cancel, true, 3
	m.Update(ctrl('c'))
	if ctx.Err() == nil || m.busy {
		t.Fatal("Ctrl+C must cancel an active read")
	}
	m.Update(operationMsg{generation: 3, result: operationResult{Action: inspectRule}})
	if m.screen != homeScreen {
		t.Fatal("stale result changed the current screen")
	}
	_, cmd := m.Update(ctrl('c'))
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("second Ctrl+C must exit")
	}
}

func TestUIWaitsForWriteBeforeExit(t *testing.T) {
	m := testUI(t)
	m.busy, m.writing, m.generation = true, true, 5
	for _, msg := range []tea.Msg{ctrl('c'), terminalSignal{}} {
		_, cmd := m.Update(msg)
		if cmd != nil || !m.exitAfter {
			t.Fatal("an active write was interrupted")
		}
	}
	_, cmd := m.Update(operationMsg{generation: 5})
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("must exit after the operation finishes")
	}
}

func TestUIPaletteSearchAndEscapePreserveForm(t *testing.T) {
	m := testUI(t)
	m.openForm("create", "")
	setField(t, m, "id", "keep-my-input")
	m.Update(ctrl('p'))
	if !m.palette {
		t.Fatal("palette did not open")
	}
	m.paletteQuery.SetValue("registry")
	items := m.paletteItems()
	if len(items) != 1 || items[0].target != "settings" {
		t.Fatalf("unexpected actions: %#v", items)
	}
	m.Update(keyPress(tea.KeyEscape))
	if m.palette || m.screen != formScreen || m.formValue("id") != "keep-my-input" {
		t.Fatal("closing palette discarded the form")
	}
}

func TestUISettingsAreSessionOnlyAndValidateBeforeCommit(t *testing.T) {
	m := testUI(t)
	m.openForm("settings", "")
	project := t.TempDir()
	before := m.backend.options
	setField(t, m, "project", project)
	setField(t, m, "ttl", "invalid")
	m.reviewForm()
	if m.project == project || m.backend.options != before {
		t.Fatal("invalid settings were partially applied")
	}
	setField(t, m, "ttl", "30m")
	m.reviewForm()
	if m.screen != homeScreen || m.project != project {
		t.Fatal("settings were not applied")
	}
	entries, err := os.ReadDir(project)
	if err != nil || len(entries) != 0 {
		t.Fatal("session settings wrote project files")
	}
}

func TestUILayoutAndLongContentAtSupportedWidths(t *testing.T) {
	for _, width := range []int{60, 80, 120} {
		t.Run(strconv.Itoa(width), func(t *testing.T) {
			m := testUI(t)
			m.resize(width, 32)
			assertFits := func() {
				t.Helper()
				view := m.View().Content
				if lipgloss.Height(view) > m.height {
					t.Fatalf("screen exceeds terminal height: %d", lipgloss.Height(view))
				}
				for _, line := range strings.Split(view, "\n") {
					if lipgloss.Width(line) > width {
						t.Fatalf("line exceeds width %d: %q", width, line)
					}
				}
			}
			assertFits()
			long := strings.Repeat("A long rule with Unicode ação 界 and a path/with/no/spaces/", 30)
			m.result = operationResult{Action: inspectRule, Rule: &archrules.Rule{Definition: contracts.Rule{ID: "architecture/test@1", Name: "Long rule", Rules: []string{long, "FINAL_SENTINEL"}}}}
			m.screen = detailScreen
			m.refreshDetail()
			assertFits()
			m.viewport.GotoBottom()
			if !strings.Contains(ansi.Strip(m.View().Content), "FINAL_SENTINEL") {
				t.Fatal("long content cannot be reached by scrolling")
			}
			m.openForm("create", "")
			setField(t, m, "id", "keep-this-name")
			m.Update(tea.WindowSizeMsg{Width: width, Height: 24})
			if m.formValue("id") != "keep-this-name" {
				t.Fatal("resize lost form input")
			}
			assertFits()
			m.Update(ctrl('p'))
			assertFits()
		})
	}
}

func TestPlainModeAndRedirectedOutputStayCompatible(t *testing.T) {
	for _, args := range [][]string{{}, {"version"}, {"rules", "list"}, {"rules", "inspect", "architecture/next-modular@1"}, {"inspect", "next/page@1234"}} {
		var normal, plain, stderr bytes.Buffer
		if code := Execute(args, &normal, &stderr, "test"); code != 0 {
			t.Fatalf("normal: %s", &stderr)
		}
		if code := Execute(append([]string{"--plain"}, args...), &plain, &stderr, "test"); code != 0 {
			t.Fatalf("plain: %s", &stderr)
		}
		if normal.String() != plain.String() || strings.Contains(plain.String(), "\x1b[") {
			t.Fatalf("plain output changed for %v", args)
		}
	}
	for _, env := range []struct{ key, value string }{{"NO_COLOR", "1"}, {"TERM", "dumb"}} {
		t.Setenv(env.key, env.value)
		var out, stderr bytes.Buffer
		if Execute(nil, &out, &stderr, "test") != 0 || strings.Contains(out.String(), "\x1b[") {
			t.Fatal("non-interactive fallback emitted ANSI")
		}
	}
}

func TestRichRenderingPreservesContentAndStripsTerminalEscapes(t *testing.T) {
	r := operationResult{Action: inspectRule, Rule: &archrules.Rule{Definition: contracts.Rule{ID: "architecture/test@1", Name: "\x1b]0;malicious\aSafe title", Rules: []string{"Do not truncate this rule."}}}}
	text := renderResult(r, newTheme(false), 60)
	if strings.Contains(text, "\x1b") || strings.Contains(text, "malicious") || !strings.Contains(text, "Safe title") {
		t.Fatalf("unsafe terminal content: %q", text)
	}
	if !strings.Contains(text, "Do not truncate this rule.") {
		t.Fatal("rule text was lost")
	}
	m := testUI(t)
	m.showMessage("Failure", strings.Repeat("Detailed error\n", 30)+"ERROR_END")
	m.viewport.GotoBottom()
	if !strings.Contains(ansi.Strip(m.View().Content), "ERROR_END") {
		t.Fatal("long errors cannot be read")
	}
	m.result.Warnings = []error{errors.New("stale registry")}
	if !strings.Contains(m.warningContent(m.result.Warnings), "stale registry") {
		t.Fatal("missing warning")
	}
}
