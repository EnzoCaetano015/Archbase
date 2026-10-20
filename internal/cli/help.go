package cli

import (
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func installHelp(root *cobra.Command, version string) {
	fallback := root.HelpFunc()
	root.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if !richCommand(cmd) {
			fallback(cmd, args)
			return
		}
		t := newTheme(os.Getenv("NO_COLOR") == "")
		var b strings.Builder
		fmt.Fprintf(&b, "\n%s %s · %s\n", t.title.Render("Archbase"), t.muted.Render("arc "+safeText(version)), cmd.Short)
		b.WriteString(t.section("Usage", -1) + "  " + cmd.UseLine() + "\n")
		if cmd != root && cmd.Long != "" {
			b.WriteString("\n" + cmd.Long + "\n")
		}
		if cmd.HasAvailableSubCommands() {
			b.WriteString(t.section("Commands", -1))
			for _, child := range cmd.Commands() {
				if child.Hidden || !child.IsAvailableCommand() {
					continue
				}
				if commandWidth(cmd) >= 95 {
					b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
						lipgloss.NewStyle().Width(42).Render("  "+t.accent.Render(child.Use)),
						lipgloss.NewStyle().Width(commandWidth(cmd)-42).Render(child.Short)) + "\n")
				} else {
					fmt.Fprintf(&b, "  %s\n    %s\n", t.accent.Render(child.Use), child.Short)
				}
			}
		}
		if cmd.HasAvailableLocalFlags() {
			b.WriteString(t.section("Flags", -1) + renderHelpFlags(cmd.LocalFlags(), t))
		}
		if cmd.HasAvailableInheritedFlags() {
			b.WriteString(t.section("Global flags", -1) + renderHelpFlags(cmd.InheritedFlags(), t))
		}
		b.WriteString("\n" + t.muted.Render("Run ") + t.title.Render(cmd.CommandPath()+" <command> --help") + t.muted.Render(" for details.") + "\n")
		fmt.Fprintln(cmd.OutOrStdout(), ansi.Wrap(b.String(), commandWidth(cmd), ""))
	})
}

func renderHelpFlags(flags *pflag.FlagSet, t theme) string {
	var b strings.Builder
	flags.VisitAll(func(flag *pflag.Flag) {
		if flag.Hidden {
			return
		}
		label := "--" + flag.Name
		if flag.Shorthand != "" {
			label = "-" + flag.Shorthand + ", " + label
		}
		if flag.Value.Type() != "bool" {
			label += " <" + flag.Value.Type() + ">"
		}
		fmt.Fprintf(&b, "  %s\n    %s", t.accent.Render(label), flag.Usage)
		if flag.DefValue != "" && flag.DefValue != "false" {
			b.WriteString(t.muted.Render(" (default: " + safeText(flag.DefValue) + ")"))
		}
		b.WriteByte('\n')
	})
	return b.String()
}
