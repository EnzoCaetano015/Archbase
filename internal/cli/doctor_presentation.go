package cli

import (
	"fmt"
	"strings"

	"github.com/EnzoCaetano015/Archbase/internal/diagnostics"
	"github.com/charmbracelet/x/ansi"
)

func renderDoctor(result diagnostics.Result, verbose bool, t theme, width int) string {
	order := []struct{ id, label string }{
		{"cli-version", "Archbase CLI"}, {"project-root", "Project directory"},
		{"workspace-index", "Workspace"}, {"scope-paths", "Scope paths"},
		{"local-bundles", "Local patterns"}, {"scope-origin", "Pattern origins"},
		{"rule-documents", "Rule documents"}, {"managed-indices", "Agent indices"},
		{"legacy-indices", "Legacy configuration"}, {"snapshot", "Filesystem snapshot"},
	}
	var b strings.Builder
	fmt.Fprintln(&b, t.title.Render("Archbase doctor")+t.muted.Render(" (use -v for details)"))
	counts := map[string]int{}
	checks := map[string]diagnostics.Check{}
	for _, c := range result.Checks {
		checks[c.ID] = c
		counts[c.Status]++
	}
	for _, item := range order {
		c, ok := checks["doctor."+item.id]
		if !ok {
			continue
		}
		label := item.label
		if item.id == "cli-version" {
			label += " (" + safeText(result.ToolVersion) + ")"
		}
		line := statusSymbol(c.Status, t) + " " + label
		if c.Status != "passed" {
			message := c.Message
			switch c.Reason {
			case "missing-workspace", "unconfigured-workspace":
				message = "Not configured."
			case "no-installed-rules":
				message = "None installed."
			case "no-active-indices":
				message = "None active."
			case "no-local-bundles":
				message = "None configured."
			case "no-declared-origin":
				message = "None declared."
			}
			line += " — " + safeText(message)
		}
		fmt.Fprintln(&b, line)
		if c.Status != "passed" && c.Path != nil {
			fmt.Fprintln(&b, "    "+safeText(*c.Path))
		}
		if !verbose && c.Status == "not-run" {
			// Coverage failures must retain their cause even in the compact view.
			reason := strings.ReplaceAll(c.Reason, "-", " ")
			if reason != "" {
				fmt.Fprintln(&b, "    "+safeText(reason))
			}
		}
		if verbose {
			fmt.Fprintf(&b, "    %s [%s]\n    %s\n", safeText(c.ID), safeText(c.Status), safeText(c.Message))
			if c.Path != nil && c.Status == "passed" {
				fmt.Fprintln(&b, "    "+safeText(*c.Path))
			}
			if c.Reason != "" {
				fmt.Fprintln(&b, "    Reason: "+safeText(c.Reason))
			}
			if c.Recommendation != "" {
				fmt.Fprintln(&b, "    Next: "+safeText(c.Recommendation))
			}
		}
	}
	state := result.State
	if result.ExitCode() == 130 {
		state = "cancelled (incomplete)"
	} else if state == "partial" {
		state = "partial (incomplete)"
	}
	summary := []string{safeText(state)}
	for _, entry := range []struct{ status, label string }{{"passed", "passed"}, {"warning", "warning"}, {"error", "error"}, {"not-applicable", "not applicable"}, {"not-run", "not run"}} {
		count := counts[entry.status]
		if count == 0 {
			continue
		}
		label := entry.label
		if count != 1 && (entry.status == "warning" || entry.status == "error") {
			label += "s"
		}
		summary = append(summary, fmt.Sprintf("%d %s", count, label))
	}
	fmt.Fprintln(&b, "\n"+strings.Join(summary, " · "))
	return ansi.Wrap(b.String(), max(1, width), "")
}
