package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/EnzoCaetano015/Archbase/internal/registry"
	archrules "github.com/EnzoCaetano015/Archbase/internal/rules"
	"github.com/charmbracelet/x/ansi"
)

type formField struct {
	id, label, hint string
	input           textinput.Model
	checkbox        bool
	checked         bool
	choices         []string
	choice          int
}

func textField(id, label, value, hint string) formField {
	input := textinput.New()
	input.Prompt = ""
	input.SetVirtualCursor(true)
	input.SetValue(value)
	return formField{id: id, label: label, hint: hint, input: input}
}

func checkField(id, label, hint string) formField {
	return formField{id: id, label: label, hint: hint, checkbox: true}
}
func choiceField(id, label string, choices ...string) formField {
	return formField{id: id, label: label, choices: choices}
}

func (f formField) value() string {
	if f.checkbox {
		return strconv.FormatBool(f.checked)
	}
	if len(f.choices) > 0 {
		return f.choices[f.choice]
	}
	return f.input.Value()
}

func (m *uiModel) openForm(target, id string) (tea.Model, tea.Cmd) {
	m.formReturn = m.screen
	if m.screen == formScreen || m.screen == confirmScreen {
		m.formReturn = homeScreen
	}
	m.formAction, m.screen, m.focus = target, formScreen, 0
	m.err, m.notice, m.fields = "", "", nil
	path := func(label string) formField { return textField("path", label, ".", "Relative to "+m.project) }
	switch target {
	case "add":
		m.fields = []formField{textField("id", "Pattern ID", id, "Example: next/page@1234"), path("Scope path")}
	case "create":
		m.fields = []formField{textField("id", "Local name", "", "Example: pages-standard"), path("Scope path"), textField("from", "Derive from (optional)", "", "Leave empty to create a new local pattern")}
	case "resolve":
		m.fields = []formField{path("Target path")}
	case "inspect":
		m.fields = []formField{textField("id", "Path or pattern ID", ".", "A local path or a canonical pattern ID")}
	case "rules add":
		m.fields = []formField{textField("id", "Rule ID", id, "Example: architecture/next-modular@1"), choiceField("format", "Export format", "cursor", "copilot", "agents"), path("Destination project"), checkField("overwrite", "Overwrite an existing Cursor/Copilot export", "Explicit opt-in; disabled by default"), checkField("merge", "Merge the managed block into AGENTS.md", "Explicit opt-in; preserves external content")}
	case "remove":
		m.fields = []formField{textField("id", "Rule ID", id, "Leave empty only when removing all managed exports"), path("Destination project"), checkField("all", "Remove ALL managed rule exports", "Only Archbase-owned files and blocks may be removed")}
	case "settings":
		o := m.backend.options
		m.fields = []formField{textField("project", "Project directory", m.project, "An existing directory; changes apply only to this session"), textField("url", "Registry URL (optional)", o.URL, "Public https, git or file URL; blank uses the built-in catalog"), textField("ref", "Registry branch or tag", o.Ref, ""), textField("subdir", "Registry subdirectory", o.Subdirectory, ""), textField("cache", "Registry cache directory", o.CacheRoot, "Absolute path"), textField("ttl", "Cache time to live", o.TTL.String(), "Example: 15m")}
	default:
		m.screen = homeScreen
		return m, nil
	}
	m.resize(m.width, m.height)
	for i := range m.fields {
		m.fields[i].input.SetStyles(m.inputStyles())
	}
	return m, m.focusField(0)
}

func (m *uiModel) inputStyles() textinput.Styles {
	state := textinput.StyleState{Prompt: m.theme.accent, Placeholder: m.theme.muted, Suggestion: m.theme.muted}
	return textinput.Styles{Focused: state, Blurred: state}
}

func (m *uiModel) focusField(index int) tea.Cmd {
	for i := range m.fields {
		m.fields[i].input.Blur()
	}
	m.focus = (index + len(m.fields) + 1) % (len(m.fields) + 1)
	if m.focus < len(m.fields) && !m.fields[m.focus].checkbox && len(m.fields[m.focus].choices) == 0 {
		return m.fields[m.focus].input.Focus()
	}
	return nil
}

func (m *uiModel) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "tab":
			return m, m.focusField(m.focus + 1)
		case "shift+tab":
			return m, m.focusField(m.focus - 1)
		case "ctrl+s":
			return m.reviewForm()
		case "enter":
			if m.focus == len(m.fields) {
				return m.reviewForm()
			}
			field := &m.fields[m.focus]
			if field.checkbox {
				field.checked = !field.checked
				return m, nil
			}
			if len(field.choices) > 0 {
				field.choice = (field.choice + 1) % len(field.choices)
				return m, nil
			}
			return m, m.focusField(m.focus + 1)
		case "left", "right", " ", "space":
			if m.focus < len(m.fields) {
				field := &m.fields[m.focus]
				if field.checkbox {
					field.checked = !field.checked
					return m, nil
				}
				if len(field.choices) > 0 {
					delta := 1
					if key.String() == "left" {
						delta = len(field.choices) - 1
					}
					field.choice = (field.choice + delta) % len(field.choices)
					return m, nil
				}
			}
		}
	}
	if m.focus < len(m.fields) {
		field := &m.fields[m.focus]
		if !field.checkbox && len(field.choices) == 0 {
			var cmd tea.Cmd
			field.input, cmd = field.input.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m *uiModel) formValue(id string) string {
	for _, field := range m.fields {
		if field.id == id {
			return field.value()
		}
	}
	return ""
}

func (m *uiModel) buildRequest() (request, error) {
	r := request{Action: action(m.formAction), ID: strings.TrimSpace(m.formValue("id")), Path: m.projectPath(m.formValue("path")), From: strings.TrimSpace(m.formValue("from")), Format: m.formValue("format"), All: m.formValue("all") == "true", Overwrite: m.formValue("overwrite") == "true", Merge: m.formValue("merge") == "true"}
	if r.Action != resolvePath && r.ID == "" && !(r.Action == removeRule && r.All) {
		return r, fmt.Errorf("an ID or local name is required")
	}
	switch r.Action {
	case addPattern:
		if _, err := registry.ParsePatternID(r.ID); err != nil {
			return r, err
		}
	case createPattern:
		if _, err := registry.ParsePatternID("local/" + r.ID + "@1"); err != nil {
			return r, fmt.Errorf("invalid local name: %w", err)
		}
		if r.From != "" {
			if _, err := registry.ParsePatternID(r.From); err != nil {
				return r, err
			}
		}
	case exportRule:
		if _, err := archrules.ParseRuleID(r.ID); err != nil {
			return r, err
		}
		if r.Format == "agents" && r.Overwrite {
			return r, fmt.Errorf("AGENTS exports use Merge, not Overwrite")
		}
		if r.Format != "agents" && r.Merge {
			return r, fmt.Errorf("Merge applies only to AGENTS exports")
		}
	case removeRule:
		if r.All && r.ID != "" {
			return r, fmt.Errorf("clear the Rule ID to remove all managed exports")
		}
		if !r.All {
			if _, err := archrules.ParseRuleID(r.ID); err != nil {
				return r, err
			}
		}
	case inspectPattern:
		if _, err := registry.ParsePatternID(r.ID); err != nil {
			r.ID = m.projectPath(r.ID)
		}
	}
	return r, nil
}

func (m *uiModel) reviewForm() (tea.Model, tea.Cmd) {
	m.err, m.notice = "", ""
	if m.formAction == "settings" {
		if err := m.saveSettings(); err != nil {
			m.err = safeText(err.Error())
			return m, nil
		}
		m.screen, m.notice = homeScreen, "Session settings updated."
		return m, nil
	}
	r, err := m.buildRequest()
	if err != nil {
		m.err = safeText(err.Error())
		return m, nil
	}
	if r.writes() {
		m.pending, m.screen = r, confirmScreen
		m.review.SetContent(ansi.Wrap(m.confirmView(), max(1, m.width-4), ""))
		m.review.GotoTop()
		return m, nil
	}
	return m, m.start(r)
}

func (m *uiModel) saveSettings() error {
	project := m.projectPath(m.formValue("project"))
	stat, err := os.Stat(project)
	if err != nil {
		return err
	}
	if !stat.IsDir() {
		return fmt.Errorf("project path must be a directory")
	}
	ttl, err := time.ParseDuration(strings.TrimSpace(m.formValue("ttl")))
	if err != nil || ttl <= 0 {
		return fmt.Errorf("cache time to live must be a positive duration")
	}
	cache := m.formValue("cache")
	if !filepath.IsAbs(cache) {
		return fmt.Errorf("registry cache directory must be absolute")
	}
	options := RegistryOptions{URL: strings.TrimSpace(m.formValue("url")), Ref: strings.TrimSpace(m.formValue("ref")), Subdirectory: strings.TrimSpace(m.formValue("subdir")), CacheRoot: filepath.Clean(cache), TTL: ttl}
	if options.URL != "" {
		// Validate syntax without connecting or changing the registry cache.
		if _, err := registry.NewGitSource(registry.GitSourceConfig{URL: options.URL, Ref: options.Ref, Subdirectory: options.Subdirectory, CacheRoot: options.CacheRoot, TTL: options.TTL}); err != nil {
			return err
		}
	}
	m.project, m.backend.options = project, options
	m.backend.options.ProjectRoot = project
	m.entries = nil
	return nil
}
