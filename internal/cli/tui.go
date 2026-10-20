package cli

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
	"github.com/spf13/cobra"
)

type screen uint8

const (
	homeScreen screen = iota
	catalogScreen
	detailScreen
	formScreen
	confirmScreen
	mcpScreen
	messageScreen
)

type menuItem struct{ name, description, target string }

var uiMenu = []menuItem{
	{"Patterns", "Explore structural patterns", "patterns"},
	{"Rules", "Explore architecture rules", "rules"},
	{"Resolve", "Find the active scope for a path", "resolve"},
	{"Create", "Create a customizable local pattern", "create"},
	{"Inspect local", "Inspect a path or pattern ID", "inspect"},
	{"Install pattern", "Install a pattern in a scope", "add"},
	{"Export rule", "Export to Cursor, Copilot or AGENTS", "rules add"},
	{"Remove exports", "Remove Archbase-owned rule exports", "remove"},
	{"Project & registry", "Change settings for this session", "settings"},
	{"MCP connection", "Connect an agent over stdio", "mcp"},
	{"Quit", "Return to your shell", "quit"},
}

type catalogEntry struct{ id, name, version, source, description string }
type operationMsg struct {
	generation uint64
	result     operationResult
	err        error
}
type terminalSignal struct{}

type uiModel struct {
	ctx                      context.Context
	backend                  operations
	version, project         string
	theme                    theme
	color                    bool
	width, height            int
	screen                   screen
	detailReturn             screen
	homeSelected             int
	catalog                  action
	entries                  []catalogEntry
	selected                 int
	query                    textinput.Model
	searching                bool
	palette                  bool
	paletteQuery             textinput.Model
	paletteSelected          int
	viewport                 viewport.Model
	review                   viewport.Model
	message                  string
	messageReturn            screen
	spinner                  spinner.Model
	result                   operationResult
	fields                   []formField
	focus                    int
	formAction               string
	formReturn               screen
	pending                  request
	busy, writing, exitAfter bool
	generation               uint64
	cancel                   context.CancelFunc
	operationDone            <-chan struct{}
	err, notice              string
	exitErr                  error
}

func newUI(ctx context.Context, backend operations, version, project string, color bool) *uiModel {
	backend.options.ProjectRoot = project
	query := textinput.New()
	query.Prompt = "Search  "
	query.Placeholder = "ID, name or description"
	query.SetVirtualCursor(true)
	palette := textinput.New()
	palette.Prompt = "› "
	palette.Placeholder = "Find an action…"
	palette.SetVirtualCursor(true)
	s := spinner.New()
	s.Spinner = spinner.Dot
	m := &uiModel{ctx: ctx, backend: backend, version: version, project: project,
		color: color, theme: newTheme(color), width: 80, height: 24,
		query: query, paletteQuery: palette, viewport: viewport.New(), review: viewport.New(), spinner: s}
	m.query.SetStyles(m.inputStyles())
	m.paletteQuery.SetStyles(m.inputStyles())
	m.resize(80, 24)
	return m
}

func runInteractive(cmd *cobra.Command, deps Dependencies, options RegistryOptions, version string) error {
	project, err := os.Getwd()
	if err != nil {
		return err
	}
	if options.ProjectRoot != "" {
		project, err = filepath.Abs(options.ProjectRoot)
		if err != nil {
			return err
		}
	}
	m := newUI(cmd.Context(), operations{deps, options}, version, project, os.Getenv("NO_COLOR") == "")
	opts := []tea.ProgramOption{tea.WithInput(cmd.InOrStdin()), tea.WithOutput(cmd.OutOrStdout()), tea.WithoutSignalHandler()}
	if !m.color {
		opts = append(opts, tea.WithColorProfile(colorprofile.Ascii))
	}
	p := tea.NewProgram(m, opts...)
	// Route signals through Update so an in-flight filesystem transaction is
	// allowed to finish (or roll back) before the terminal is released.
	signals := make(chan os.Signal, 1)
	done := make(chan struct{})
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	defer close(done)
	go func() {
		for {
			select {
			case <-signals:
				p.Send(terminalSignal{})
			case <-done:
				return
			}
		}
	}()
	_, err = p.Run()
	if m.cancel != nil {
		m.cancel()
	}
	if m.writing && m.operationDone != nil {
		<-m.operationDone
	}
	if err == nil {
		err = m.exitErr
	}
	return err
}

func (m *uiModel) Init() tea.Cmd { return nil }

func (m *uiModel) resize(width, height int) {
	m.width, m.height = max(1, width), max(1, height)
	m.viewport.SetWidth(max(1, width-4))
	m.viewport.SetHeight(max(1, height-8))
	m.review.SetWidth(max(1, width-4))
	m.review.SetHeight(max(1, height-8))
	m.query.SetWidth(max(1, width-14))
	m.paletteQuery.SetWidth(max(1, width-10))
	for i := range m.fields {
		m.fields[i].input.SetWidth(max(1, width-12))
	}
	if m.screen == detailScreen {
		m.refreshDetail()
	}
	if m.screen == confirmScreen {
		m.review.SetContent(ansi.Wrap(m.confirmView(), max(1, width-4), ""))
	}
	if m.screen == messageScreen {
		m.viewport.SetContent(ansi.Wrap(m.message, max(1, width-4), ""))
	}
	if m.screen == mcpScreen {
		m.viewport.SetContent(ansi.Wrap(m.mcpContent(), max(1, width-4), ""))
	}
}

func (m *uiModel) refreshDetail() {
	position := m.viewport.YOffset()
	content := renderResult(m.result, m.theme, max(1, m.width-4))
	if len(m.result.Warnings) > 0 {
		content += "\n" + m.warningContent(m.result.Warnings)
	}
	m.viewport.SetContent(content)
	m.viewport.SetYOffset(position)
}

func (m *uiModel) stopRead() {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	m.generation++
	m.busy = false
}

func (m *uiModel) start(r request) tea.Cmd {
	if r.Action != listPatterns && r.Action != listRules {
		m.detailReturn = homeScreen
		if m.screen == catalogScreen || (m.formReturn == detailScreen && len(m.entries) > 0) {
			m.detailReturn = catalogScreen
		}
	}
	m.stopRead()
	m.generation++
	generation := m.generation
	ctx, cancel := context.WithCancel(m.ctx)
	if r.writes() {
		// Once confirmed, writes are not interrupted mid-transaction.
		ctx = context.WithoutCancel(m.ctx)
	}
	m.cancel = cancel
	m.busy, m.writing, m.err, m.notice = true, r.writes(), "", ""
	backend := m.backend // Snapshot session settings for this request.
	done := make(chan struct{})
	m.operationDone = done
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		defer close(done)
		result, err := backend.run(ctx, r)
		return operationMsg{generation, result, err}
	})
}

func (m *uiModel) exit() (tea.Model, tea.Cmd) {
	if m.busy && m.writing {
		m.exitAfter = true
		m.notice = "Finishing the filesystem operation before exiting…"
		return m, nil
	}
	m.stopRead()
	return m, tea.Quit
}

func (m *uiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
		return m, nil
	case terminalSignal:
		return m.exit()
	case operationMsg:
		if msg.generation != m.generation {
			return m, nil
		}
		m.busy, m.writing = false, false
		if m.cancel != nil {
			m.cancel()
			m.cancel = nil
		}
		if m.exitAfter {
			m.exitErr = msg.err
			return m, tea.Quit
		}
		if msg.err != nil {
			if m.screen == confirmScreen {
				m.screen = formScreen
			}
			m.showMessage("Operation failed", safeText(msg.err.Error())+"\n\n"+m.warningContent(msg.result.Warnings))
			return m, nil
		}
		m.result = msg.result
		m.err = ""
		if msg.result.Action == listPatterns || msg.result.Action == listRules {
			m.entries = nil
			for _, e := range msg.result.Patterns {
				m.entries = append(m.entries, catalogEntry{e.ID.String(), msg.result.Names[e.ID.String()], e.Version, e.Source, e.Description})
			}
			for _, e := range msg.result.Rules {
				m.entries = append(m.entries, catalogEntry{e.ID.String(), msg.result.Names[e.ID.String()], e.Version, e.Source, e.Description})
			}
			m.selected = 0
		} else {
			m.screen = detailScreen
			m.viewport.GotoTop()
			m.refreshDetail()
		}
		return m, nil
	case spinner.TickMsg:
		if m.busy {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	case tea.KeyPressMsg:
		key := msg.String()
		if key == "ctrl+c" {
			if m.busy && !m.writing {
				m.stopRead()
				m.notice = "Query cancelled. Press Esc to go back."
				return m, nil
			}
			return m.exit()
		}
		if m.busy && m.writing {
			return m, nil
		}
		if key == "ctrl+p" {
			m.palette = !m.palette
			m.paletteSelected = 0
			m.paletteQuery.SetValue("")
			if m.palette {
				return m, m.paletteQuery.Focus()
			}
			return m, nil
		}
		if m.palette {
			return m.updatePalette(msg)
		}
		if key == "esc" {
			return m.back()
		}
		if m.busy {
			return m, nil
		}
		switch m.screen {
		case homeScreen:
			switch key {
			case "up", "k":
				m.homeSelected = max(0, m.homeSelected-1)
			case "down", "j":
				m.homeSelected = min(len(uiMenu)-1, m.homeSelected+1)
			case "enter":
				return m.activate(uiMenu[m.homeSelected].target)
			case "p":
				return m.activate("patterns")
			case "r":
				return m.activate("rules")
			case "q":
				return m.exit()
			}
		case catalogScreen:
			if key == "w" && !m.searching {
				m.showMessage("Warnings", m.warningContent(m.result.Warnings))
				return m, nil
			}
			return m.updateCatalog(msg)
		case formScreen:
			return m.updateForm(msg)
		case confirmScreen:
			if key == "enter" {
				return m, m.start(m.pending)
			}
			var cmd tea.Cmd
			m.review, cmd = m.review.Update(msg)
			return m, cmd
		case detailScreen:
			if key == "a" && m.result.Pattern != nil {
				return m.openForm("add", m.result.Pattern.Entry.ID.String())
			}
			if key == "e" && m.result.Rule != nil {
				return m.openForm("rules add", m.result.Rule.Definition.ID)
			}
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		case mcpScreen, messageScreen:
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
	}
	// Bubble text inputs also receive paste and virtual-cursor events.
	if m.palette {
		return m.updatePalette(msg)
	}
	if m.screen == formScreen && !m.busy {
		return m.updateForm(msg)
	}
	if m.screen == catalogScreen && m.searching {
		return m.updateCatalog(msg)
	}
	return m, nil
}

func (m *uiModel) back() (tea.Model, tea.Cmd) {
	if m.busy {
		m.stopRead()
	}
	m.err, m.notice = "", ""
	switch m.screen {
	case messageScreen:
		m.screen = m.messageReturn
		if m.screen == detailScreen {
			m.refreshDetail()
		}
	case confirmScreen:
		m.screen = formScreen
	case formScreen:
		m.screen = m.formReturn
		if m.screen == detailScreen {
			m.refreshDetail()
		}
	case detailScreen:
		m.screen = m.detailReturn
	case catalogScreen:
		if m.searching {
			m.searching = false
			m.query.Blur()
		} else {
			m.screen = homeScreen
		}
	default:
		m.screen = homeScreen
	}
	return m, nil
}

func (m *uiModel) activate(target string) (tea.Model, tea.Cmd) {
	m.palette, m.err, m.notice = false, "", ""
	if m.busy {
		m.stopRead()
	}
	switch target {
	case "patterns", "rules":
		m.catalog = action(target)
		m.screen, m.selected, m.searching = catalogScreen, 0, false
		m.query.SetValue("")
		m.query.Blur()
		m.entries = nil
		return m, m.start(request{Action: m.catalog, WithNames: true})
	case "mcp":
		m.screen = mcpScreen
		m.viewport.GotoTop()
		m.viewport.SetContent(ansi.Wrap(m.mcpContent(), max(1, m.width-4), ""))
		return m, nil
	case "quit":
		return m.exit()
	default:
		return m.openForm(target, "")
	}
}

func (m *uiModel) filteredEntries() []catalogEntry {
	query := strings.ToLower(strings.TrimSpace(m.query.Value()))
	var entries []catalogEntry
	for _, e := range m.entries {
		if strings.Contains(strings.ToLower(e.id+" "+e.name+" "+e.description), query) {
			entries = append(entries, e)
		}
	}
	return entries
}

func (m *uiModel) updateCatalog(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		entries := m.filteredEntries()
		switch key.String() {
		case "/":
			if !m.searching {
				m.searching = true
				return m, m.query.Focus()
			}
		case "up":
			m.selected = max(0, m.selected-1)
			return m, nil
		case "down":
			m.selected = max(0, min(len(entries)-1, m.selected+1))
			return m, nil
		case "pgup":
			m.selected = max(0, m.selected-5)
			return m, nil
		case "pgdown":
			m.selected = max(0, min(len(entries)-1, m.selected+5))
			return m, nil
		case "enter":
			if len(entries) == 0 {
				return m, nil
			}
			m.searching = false
			m.query.Blur()
			act := inspectPattern
			if m.catalog == listRules {
				act = inspectRule
			}
			return m, m.start(request{Action: act, ID: entries[m.selected].id})
		case "ctrl+r":
			return m, m.start(request{Action: m.catalog, WithNames: true})
		}
	}
	if m.searching {
		old := m.query.Value()
		var cmd tea.Cmd
		m.query, cmd = m.query.Update(msg)
		if old != m.query.Value() {
			m.selected = 0
		}
		return m, cmd
	}
	return m, nil
}

func (m *uiModel) paletteItems() []menuItem {
	query := strings.ToLower(m.paletteQuery.Value())
	var items []menuItem
	for _, item := range uiMenu {
		if strings.Contains(strings.ToLower(item.name+" "+item.description+" "+item.target), query) {
			items = append(items, item)
		}
	}
	return items
}

func (m *uiModel) updatePalette(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		items := m.paletteItems()
		switch key.String() {
		case "esc":
			m.palette = false
			return m, nil
		case "up":
			m.paletteSelected = max(0, m.paletteSelected-1)
			return m, nil
		case "down":
			m.paletteSelected = max(0, min(len(items)-1, m.paletteSelected+1))
			return m, nil
		case "enter":
			if len(items) > 0 {
				return m.activate(items[m.paletteSelected].target)
			}
			return m, nil
		}
	}
	old := m.paletteQuery.Value()
	var cmd tea.Cmd
	m.paletteQuery, cmd = m.paletteQuery.Update(msg)
	if old != m.paletteQuery.Value() {
		m.paletteSelected = 0
	}
	return m, cmd
}

func (m *uiModel) projectPath(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(m.project, path)
}

func (m *uiModel) mcpContent() string {
	return "MCP CONNECTION\n\nAdd this configuration to your agent:\n\n" + m.connectionJSON() + "\n\nThe MCP server runs as a separate process.\nRead-only tools stay scoped to the configured project root.\n\nSee docs/mcp.md for client configuration.\n"
}

func (m *uiModel) showMessage(title, body string) {
	m.messageReturn, m.screen = m.screen, messageScreen
	m.message = m.theme.title.Render(title) + "\n\n" + body
	m.viewport.SetContent(ansi.Wrap(m.message, max(1, m.width-4), ""))
	m.viewport.GotoTop()
}

func (m *uiModel) warningContent(warnings []error) string {
	if len(warnings) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(m.theme.muted.Render("WARNINGS") + "\n\n")
	for _, warning := range warnings {
		if warning != nil {
			b.WriteString(safeText(warning.Error()) + "\n\n")
		}
	}
	return b.String()
}
