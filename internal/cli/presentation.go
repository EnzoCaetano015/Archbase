package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"
)

type theme struct{ title, muted, accent, success, danger, border lipgloss.Style }

func newTheme(color bool) theme {
	t := theme{}
	if color {
		t.title = lipgloss.NewStyle().Foreground(lipgloss.Color("#E4E6E9")).Bold(true)
		t.muted = lipgloss.NewStyle().Foreground(lipgloss.Color("#919BA5"))
		t.accent = lipgloss.NewStyle().Foreground(lipgloss.Color("#5EEAD4"))
		t.success = lipgloss.NewStyle().Foreground(lipgloss.Color("#86EFAC"))
		t.danger = lipgloss.NewStyle().Foreground(lipgloss.Color("#FDA4AF"))
		t.border = lipgloss.NewStyle().Foreground(lipgloss.Color("#394148"))
	}
	return t
}

// Registry descriptions and paths are content, never terminal control sequences.
func safeText(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' || !unicode.IsControl(r) {
			return r
		}
		return -1
	}, ansi.Strip(s))
}

func terminalStream(value any) bool {
	file, ok := value.(*os.File)
	return ok && term.IsTerminal(file.Fd())
}

func plainCommand(cmd *cobra.Command) bool {
	plain, _ := cmd.Root().PersistentFlags().GetBool("plain")
	return plain || os.Getenv("TERM") == "dumb"
}

func richCommand(cmd *cobra.Command) bool {
	return !plainCommand(cmd) && terminalStream(cmd.OutOrStdout())
}

func commandWidth(cmd *cobra.Command) int {
	if file, ok := cmd.OutOrStdout().(*os.File); ok {
		if width, _, err := term.GetSize(file.Fd()); err == nil {
			return max(20, width-2)
		}
	}
	return 78
}

func executeOperation(cmd *cobra.Command, deps Dependencies, options RegistryOptions, r request) error {
	result, err := (operations{deps, options}).run(cmd.Context(), r)
	if richCommand(cmd) && terminalStream(cmd.ErrOrStderr()) {
		t := newTheme(os.Getenv("NO_COLOR") == "")
		for _, warning := range result.Warnings {
			if warning != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), t.muted.Render("Warning  ")+safeText(warning.Error()))
			}
		}
	} else {
		printWarnings(cmd.ErrOrStderr(), result.Warnings)
	}
	if err != nil {
		return err
	}
	if richCommand(cmd) {
		_, err = fmt.Fprintln(cmd.OutOrStdout(), "\n"+renderResult(result, newTheme(os.Getenv("NO_COLOR") == ""), commandWidth(cmd))+"\n")
		return err
	}
	return printPlainResult(cmd.OutOrStdout(), result)
}

// Retain the existing non-interactive output contract, including field names.
func printPlainResult(out io.Writer, r operationResult) error {
	switch r.Action {
	case listRules:
		w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tVERSION\tSOURCE\tDESCRIPTION")
		for _, e := range r.Rules {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", e.ID, e.Version, e.Source, e.Description)
		}
		return w.Flush()
	case inspectRule:
		printRule(out, *r.Rule)
	case inspectPattern:
		if r.Resolution != nil {
			printPattern(out, *r.Pattern, r.Resolution.Scope.Pattern.Source, r.Resolution.Scope.Origin)
		} else {
			printPattern(out, *r.Pattern, r.Pattern.Source, nil)
		}
	case resolvePath:
		printResolution(out, *r.Resolution)
	case addPattern, createPattern:
		verb := "Installed"
		if r.Action == createPattern {
			verb = "Created"
		}
		fmt.Fprintf(out, "%s: %s\nScope: %s\nPattern root: %s\n", verb, r.Installed.PatternID, r.Installed.ScopeDirectory, r.Installed.PatternDirectory)
	case exportRule:
		fmt.Fprintf(out, "Exported: %s\nFormat: %s\nFiles:\n", r.ID, r.Exported.Format)
		for _, path := range r.Exported.Paths {
			fmt.Fprintf(out, "  %s\n", path)
		}
	case removeRule:
		if len(r.Removed.Paths) == 0 {
			if r.ID == "" {
				fmt.Fprintln(out, "No managed rule exports found.")
			} else {
				fmt.Fprintf(out, "No managed exports found for: %s\n", r.ID)
			}
			return nil
		}
		if len(r.Removed.RuleIDs) > 0 {
			fmt.Fprintln(out, "Removed rules:")
			for _, id := range r.Removed.RuleIDs {
				fmt.Fprintf(out, "  %s\n", id)
			}
		}
		if r.Removed.LegacyExports > 0 {
			fmt.Fprintf(out, "Legacy exports: %d\n", r.Removed.LegacyExports)
		}
		fmt.Fprintln(out, "Files:")
		for _, path := range r.Removed.Paths {
			fmt.Fprintf(out, "  %s\n", path)
		}
	}
	return nil
}

func (t theme) section(label string, count int) string {
	if count >= 0 {
		label = fmt.Sprintf("%s · %d", label, count)
	}
	return "\n" + t.muted.Render(strings.ToUpper(label)) + "\n"
}

func renderResult(r operationResult, t theme, width int) string {
	var b strings.Builder
	line := func(s string) { b.WriteString(s + "\n") }
	meta := func(k, v string) { line(t.muted.Render(k+"  ") + safeText(v)) }
	header := func(name, id, version, source, description string) {
		line(t.title.Render(safeText(name)))
		line(t.muted.Render(safeText(id)+" · v"+safeText(version)) + " · " + t.accent.Render(safeText(source)))
		if description != "" {
			line("\n" + safeText(description))
		}
		line(t.border.Render(strings.Repeat("─", max(1, min(width, 100)))))
	}
	switch r.Action {
	case listRules:
		line(t.title.Render(fmt.Sprintf("Architecture rules · %d", len(r.Rules))))
		if width >= 95 {
			idWidth := 32
			for _, entry := range r.Rules {
				idWidth = max(idWidth, lipgloss.Width(entry.ID.String()))
			}
			idWidth = min(idWidth, width/2)
			descWidth := max(1, width-idWidth-12)
			row := func(id, version, description string) string {
				return lipgloss.JoinHorizontal(lipgloss.Top,
					lipgloss.NewStyle().Width(idWidth+2).Render(ansi.Wrap(id, idWidth, "")),
					lipgloss.NewStyle().Width(10).Render(version),
					lipgloss.NewStyle().Width(descWidth).Render(ansi.Wrap(description, descWidth, "")))
			}
			line(t.muted.Render(row("RULE", "VER", "DESCRIPTION / SOURCE")))
			line(t.border.Render(strings.Repeat("─", width)))
			for _, entry := range r.Rules {
				line(row(t.title.Render(entry.ID.String()), t.muted.Render(safeText(entry.Version)), safeText(entry.Description)+"\n"+t.muted.Render(safeText(entry.Source))))
			}
			line("\n" + t.accent.Render("→ arc rules inspect <rule-id>"))
			break
		}
		line(t.muted.Render("RULE / VERSION / SOURCE"))
		for _, entry := range r.Rules {
			line("\n" + t.title.Render(entry.ID.String()) + "  " + t.muted.Render(safeText(entry.Version)+" · "+safeText(entry.Source)))
			line("  " + safeText(entry.Description))
		}
		line("\n" + t.accent.Render("→ arc rules inspect <rule-id>"))
	case inspectRule:
		d := r.Rule.Definition
		header(d.Name, d.ID, d.Version, r.Rule.Source, d.Description)
		line(t.section("Scopes", len(d.Scopes)))
		for _, scope := range d.Scopes {
			line("  " + safeText(scope.Path) + t.muted.Render("  →  ") + safeText(scope.Pattern))
		}
		line(t.section("Rules", len(d.Rules)))
		for i, rule := range d.Rules {
			line(t.muted.Render(fmt.Sprintf("%02d  ", i+1)) + safeText(rule))
		}
	case inspectPattern, resolvePath:
		p := r.Pattern
		if p == nil {
			p = &r.Resolution.Pattern
		}
		d := p.Bundle.Manifest
		header(d.Name, d.ID, d.Version, p.Source, d.Description)
		meta("Type", d.Type)
		if r.Resolution != nil {
			meta("Scope", r.Resolution.ScopeDirectory)
			meta("Pattern root", r.Resolution.PatternRoot)
			if origin := r.Resolution.Scope.Origin; origin != nil {
				meta("Origin", fmt.Sprintf("%s (%s, version %s)", origin.Registry, origin.ID, origin.Version))
			}
		}
		line(t.section("Files", len(p.Bundle.Files)))
		for _, f := range p.Bundle.Files {
			line("  " + safeText(f.Spec.Source) + t.muted.Render("  →  ") + safeText(f.Spec.Destination))
			line(t.muted.Render(fmt.Sprintf("    required=%t · present=%t", f.Spec.Required, f.Present)))
		}
		line(t.section("Allowed changes", len(d.AllowedChanges)))
		for _, value := range d.AllowedChanges {
			line("  • " + safeText(value))
		}
		line(t.section("Preserve", len(d.Preserve)))
		for _, value := range d.Preserve {
			line("  • " + safeText(value))
		}
	case addPattern, createPattern:
		verb := "Pattern installed"
		if r.Action == createPattern {
			verb = "Local pattern created"
		}
		line(t.success.Render("✓ " + verb))
		meta("Pattern", r.Installed.PatternID)
		meta("Scope", r.Installed.ScopeDirectory)
		meta("Pattern root", r.Installed.PatternDirectory)
	case exportRule:
		line(t.success.Render("✓ Architecture rule exported"))
		meta("Rule", r.ID)
		meta("Format", string(r.Exported.Format))
		line(t.section("Files", len(r.Exported.Paths)))
		for _, path := range r.Exported.Paths {
			line("  " + safeText(path))
		}
	case removeRule:
		if len(r.Removed.Paths) == 0 {
			line("No managed exports found.")
			break
		}
		line(t.success.Render("✓ Managed rule exports removed"))
		for _, id := range r.Removed.RuleIDs {
			meta("Rule", id)
		}
		if r.Removed.LegacyExports > 0 {
			meta("Legacy exports", fmt.Sprint(r.Removed.LegacyExports))
		}
		line(t.section("Files", len(r.Removed.Paths)))
		for _, path := range r.Removed.Paths {
			line("  " + safeText(path))
		}
	}
	return ansi.Wrap(b.String(), max(1, width), "")
}
