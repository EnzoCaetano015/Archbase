package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func (m *uiModel) View() tea.View {
	width := max(1, m.width-4)
	header := m.theme.title.Render("Archbase") + m.theme.muted.Render("  arc "+safeText(m.version)) + "  " + m.theme.accent.Render(m.screenTitle())
	header += "\n" + m.theme.muted.Render(ansi.Truncate(safeText(m.project), width, "…"))
	header += "\n" + m.theme.border.Render(strings.Repeat("─", width))
	body := ""
	if m.palette {
		body = m.paletteView()
	} else if m.busy {
		body = m.spinner.View() + "  Loading…"
		if m.writing {
			body = m.spinner.View() + "  Applying changes…\n\nPlease wait for the operation to finish."
		}
	} else {
		switch m.screen {
		case homeScreen:
			body = m.homeView()
		case catalogScreen:
			body = m.catalogView()
		case detailScreen, mcpScreen, messageScreen:
			body = m.viewport.View()
		case formScreen:
			body = m.formView()
		case confirmScreen:
			body = m.review.View()
		}
	}
	if m.err != "" {
		body = m.theme.danger.Render("Error  "+safeText(m.err)) + "\n\n" + body
	}
	if m.notice != "" {
		body = m.theme.accent.Render(safeText(m.notice)) + "\n\n" + body
	}
	body = fitBody(body, width, max(1, m.height-7))
	footer := m.footer()
	content := header + "\n" + body + "\n" + m.theme.border.Render(strings.Repeat("─", width)) + "\n" + m.theme.muted.Render(ansi.Truncate(footer, width, "…"))
	content = lipgloss.NewStyle().Padding(0, 2).Width(m.width).Render(content)
	// Keep the whole screen legible even when the terminal background is light.
	view := tea.NewView(content)
	view.AltScreen = true
	if m.color {
		view.BackgroundColor = lipgloss.Color("#0B0D0F")
		view.ForegroundColor = lipgloss.Color("#E4E6E9")
	}
	return view
}

func fitBody(body string, width, height int) string {
	lines := strings.Split(ansi.Wrap(body, width, ""), "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func (m *uiModel) screenTitle() string {
	if m.palette {
		return "Command palette"
	}
	switch m.screen {
	case catalogScreen:
		if m.catalog == listRules {
			return "Rules"
		}
		return "Patterns"
	case detailScreen:
		return "Details"
	case formScreen:
		titles := map[string]string{"add": "Install pattern", "create": "Create local pattern", "resolve": "Resolve scope", "inspect": "Inspect local", "rules add": "Export rule", "remove": "Remove managed exports", "settings": "Project & registry"}
		return titles[m.formAction]
	case confirmScreen:
		return "Review changes"
	case mcpScreen:
		return "MCP connection"
	case messageScreen:
		return "Messages"
	default:
		return "Home"
	}
}

func (m *uiModel) homeView() string {
	width := max(1, m.width-4)
	var b strings.Builder
	// The large wordmark is reserved for the home screen; narrow windows use
	// a compact brand rather than losing letters to horizontal clipping.
	if width >= 63 && m.height >= 20 {
		shades := []string{"#F4F4F5", "#E4E4E7", "#D4D4D8", "#B4B4BB", "#96969F", "#797984"}
		for row := range 6 {
			var line strings.Builder
			for col, letter := range "ARCHBASE" {
				if col > 0 {
					line.WriteByte(' ')
				}
				line.WriteString(welcomeLetters[letter][row])
			}
			word := line.String()
			if m.color {
				word = lipgloss.NewStyle().Foreground(lipgloss.Color(shades[row])).Render(word)
			}
			b.WriteString(lipgloss.PlaceHorizontal(width, lipgloss.Center, word) + "\n")
		}
		b.WriteByte('\n')
	} else {
		b.WriteString(m.theme.title.Render("ARCHBASE") + "\n\n")
	}
	if m.height >= 24 {
		b.WriteString(m.theme.muted.Render("Architecture that AI coding agents can actually follow.") + "\n\n")
	}
	available := max(1, m.height-7-strings.Count(b.String(), "\n")-1)
	start := max(0, m.homeSelected-available+1)
	for i := start; i < min(len(uiMenu), start+available); i++ {
		item := uiMenu[i]
		prefix := "  "
		label := item.name
		if i == m.homeSelected {
			prefix = "› "
			label = m.theme.accent.Render(label)
		} else {
			label = m.theme.title.Render(label)
		}
		if width >= 80 {
			label += m.theme.muted.Render("  " + item.description)
		}
		fmt.Fprintf(&b, "%s%s\n", prefix, label)
	}
	return b.String()
}

func (m *uiModel) catalogView() string {
	entries := m.filteredEntries()
	var b strings.Builder
	sources := []string{}
	seen := map[string]bool{}
	for _, entry := range m.entries {
		if !seen[entry.source] {
			sources = append(sources, entry.source)
			seen[entry.source] = true
		}
	}
	fmt.Fprintf(&b, "%s\n%s\n\n", m.query.View(), m.theme.muted.Render(fmt.Sprintf("%d of %d results · %s", len(entries), len(m.entries), safeText(strings.Join(sources, ", ")))))
	if len(entries) == 0 {
		return b.String() + "No matches. Press / to search or Ctrl+R to refresh."
	}
	width := max(1, m.width-4)
	row := func(index int) string {
		e := entries[index]
		prefix := "  "
		label := m.theme.title.Render(safeText(e.id))
		if index == m.selected {
			prefix = m.theme.accent.Render("› ")
			label = m.theme.accent.Render(safeText(e.id))
		}
		metadata := m.theme.muted.Render(safeText(e.version))
		if len(sources) > 1 {
			metadata += m.theme.muted.Render(" · " + safeText(e.source))
		}
		if width >= 100 {
			idWidth := min(48, width/2)
			descWidth := width - idWidth - 13
			return lipgloss.JoinHorizontal(lipgloss.Top,
				lipgloss.NewStyle().Width(idWidth+2).Render(ansi.Wrap(prefix+label, idWidth+2, "")),
				lipgloss.NewStyle().Width(10).Render(ansi.Wrap(metadata, 10, "")),
				lipgloss.NewStyle().Width(descWidth).Render(ansi.Wrap(safeText(e.description), descWidth, "")))
		}
		return ansi.Wrap(prefix+label+"  "+metadata+"\n  "+safeText(e.description), width, "")
	}
	if width >= 100 {
		b.WriteString(m.theme.muted.Render("ID / VERSION / DESCRIPTION") + "\n")
	}
	budget := max(1, m.height-12)
	used, start := 0, m.selected
	for start > 0 {
		n := lipgloss.Height(row(start-1)) + 1
		if used+n > budget/2 {
			break
		}
		used += n
		start--
	}
	used = 0
	for i := start; i < len(entries); i++ {
		entry := row(i)
		lines := lipgloss.Height(entry) + 1
		if used+lines > budget && i != m.selected {
			if i < m.selected {
				continue
			}
			break
		}
		b.WriteString(entry + "\n\n")
		used += lines
	}
	return b.String()
}

func (m *uiModel) paletteView() string {
	var b strings.Builder
	b.WriteString(m.theme.title.Render("What would you like to do?") + "\n\n" + m.paletteQuery.View() + "\n\n")
	items := m.paletteItems()
	if len(items) == 0 {
		return b.String() + "No matching actions."
	}
	count := max(1, m.height-13)
	start := max(0, m.paletteSelected-count+1)
	for i := start; i < min(len(items), start+count); i++ {
		prefix, label := "  ", items[i].name
		if i == m.paletteSelected {
			prefix, label = "› ", m.theme.accent.Render(label)
		}
		b.WriteString(prefix + label + "\n")
	}
	return b.String()
}

func (m *uiModel) formView() string {
	var rows []string
	focusLine := 0
	for i, field := range m.fields {
		if i == m.focus {
			focusLine = len(rows)
		}
		prefix := "  "
		label := field.label
		if i == m.focus {
			prefix = "› "
			label = m.theme.accent.Render(label)
		}
		block := prefix + label + "\n  "
		switch {
		case field.checkbox:
			if field.checked {
				block += "[x] Enabled"
			} else {
				block += "[ ] Disabled"
			}
		case len(field.choices) > 0:
			block += "‹ " + field.value() + " ›"
		default:
			block += field.input.View()
		}
		if field.hint != "" {
			block += "\n  " + m.theme.muted.Render(safeText(field.hint))
		}
		rows = append(rows, strings.Split(ansi.Wrap(block, max(1, m.width-4), ""), "\n")...)
		rows = append(rows, "")
	}
	button := "Review changes →"
	if m.formAction == "settings" {
		button = "Save session settings →"
	}
	if !(request{Action: action(m.formAction)}).writes() && m.formAction != "settings" {
		button = "Inspect →"
		if m.formAction == "resolve" {
			button = "Resolve scope →"
		}
	}
	if m.focus == len(m.fields) {
		focusLine = len(rows)
		button = m.theme.accent.Render("› " + button)
	} else {
		button = "  " + button
	}
	rows = append(rows, button)
	budget := max(1, m.height-9)
	start := max(0, focusLine-budget+4)
	start = min(start, max(0, len(rows)-budget))
	return strings.Join(rows[start:min(len(rows), start+budget)], "\n")
}

func (m *uiModel) confirmView() string {
	r := m.pending
	var b strings.Builder
	b.WriteString(m.theme.title.Render("Review before applying changes") + "\n\n")
	fmt.Fprintf(&b, "Action       %s\n", r.Action)
	if r.All {
		b.WriteString("Rule         ALL managed rule exports\n")
	} else {
		fmt.Fprintf(&b, "ID / name    %s\n", safeText(r.ID))
	}
	fmt.Fprintf(&b, "Destination  %s\n", safeText(r.Path))
	if r.From != "" {
		fmt.Fprintf(&b, "Derived from %s\n", safeText(r.From))
	}
	if r.Action == exportRule {
		fmt.Fprintf(&b, "Format       %s\nOverwrite    %t\nMerge        %t\n", r.Format, r.Overwrite, r.Merge)
	}
	if r.Action == removeRule {
		b.WriteString("\nOnly Archbase-owned exports and managed blocks will be removed.\n")
	}
	b.WriteString("\n" + m.theme.accent.Render("Enter  Apply changes") + "    " + m.theme.muted.Render("Esc  Edit"))
	return b.String()
}

func (m *uiModel) footer() string {
	if m.palette {
		return "↑↓ select · Enter open · Esc close"
	}
	if m.busy && m.writing {
		return "Changes in progress · Ctrl+C exit after completion"
	}
	if m.busy {
		return "Ctrl+C cancel query · Esc back"
	}
	switch m.screen {
	case homeScreen:
		return "↑↓ navigate · Enter open · Ctrl+P actions · q quit"
	case catalogScreen:
		if m.width < 100 {
			return "↑↓ select · / search · Enter open · Esc back · Ctrl+P"
		}
		return "↑↓ select · / search · Enter inspect · w warnings · Ctrl+R refresh · Esc back · Ctrl+P actions"
	case formScreen:
		if m.width < 80 {
			return "Tab next · Enter select · Ctrl+S review · Esc back"
		}
		return "Tab next · Shift+Tab previous · Enter select · Ctrl+S review · Esc back"
	case confirmScreen:
		return "Enter apply · Esc edit · Ctrl+P actions"
	case detailScreen:
		hint := "↑↓ / PgUp PgDn scroll · Esc back · Ctrl+P actions"
		if m.result.Pattern != nil {
			hint = "a install · " + hint
		}
		if m.result.Rule != nil {
			hint = "e export · " + hint
		}
		return hint
	default:
		return "↑↓ / PgUp PgDn scroll · Esc back · Ctrl+P actions"
	}
}

// JSON quoting keeps Windows backslashes valid in the example configuration.
func (m *uiModel) connectionJSON() string {
	data, _ := json.MarshalIndent(map[string]any{"mcpServers": map[string]any{"archbase": map[string]any{"command": "arc", "args": []string{"mcp", "serve", "--project-root", m.project}}}}, "", "  ")
	return string(data)
}
