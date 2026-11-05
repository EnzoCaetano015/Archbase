package cli

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/spf13/cobra"
)

// Only the animation runs asynchronously. Stop joins it before any final output.
func animateProgress(out io.Writer, label string, width int, delay time.Duration) func() {
	done, stopped := make(chan struct{}), make(chan struct{})
	var once sync.Once
	go func() {
		defer close(stopped)
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-done:
			return
		case <-timer.C:
		}
		tick := time.NewTicker(80 * time.Millisecond)
		defer tick.Stop()
		defer fmt.Fprint(out, "\r\x1b[2K")
		for frame := 0; ; frame++ {
			fmt.Fprint(out, "\r\x1b[2K", ansi.Truncate(string("|/-\\"[frame%4])+" "+safeText(label), max(1, width), "…"))
			select {
			case <-done:
				return
			case <-tick.C:
			}
		}
	}()
	return func() { once.Do(func() { close(done) }); <-stopped }
}

func startProgress(cmd *cobra.Command, label string) func() {
	if !richCommand(cmd) || !terminalStream(cmd.ErrOrStderr()) {
		return func() {}
	}
	file := cmd.ErrOrStderr().(*os.File)
	ok, restore := welcomeTerminal(file)
	if !ok {
		return func() {}
	}
	stop := animateProgress(file, label, commandWidth(cmd), 200*time.Millisecond)
	var once sync.Once
	return func() { once.Do(func() { stop(); restore() }) }
}

func operationLabel(a action) string {
	switch a {
	case addPattern:
		return "Installing pattern…"
	case createPattern:
		return "Creating local pattern…"
	case resolvePath:
		return "Resolving active pattern…"
	case inspectPattern:
		return "Inspecting pattern…"
	case inspectRule:
		return "Inspecting rule…"
	case listRules:
		return "Loading architecture rules…"
	case exportRule:
		return "Exporting architecture rule…"
	case removeRule:
		return "Removing managed rule exports…"
	default:
		return "Loading…"
	}
}

func statusSymbol(status string, t theme) string {
	switch status {
	case "passed":
		return t.success.Render("[✓]")
	case "warning":
		return t.accent.Render("[!]")
	case "error":
		return t.danger.Render("[✗]")
	case "not-applicable":
		return t.muted.Render("[-]")
	default:
		return t.muted.Render("[?]")
	}
}

func printCommandWarning(cmd *cobra.Command, message string) {
	if richCommand(cmd) && terminalStream(cmd.ErrOrStderr()) {
		fmt.Fprintln(cmd.ErrOrStderr(), statusSymbol("warning", newTheme(os.Getenv("NO_COLOR") == ""))+" "+safeText(message))
	} else {
		fmt.Fprintln(cmd.ErrOrStderr(), "warning: "+safeText(message))
	}
}
