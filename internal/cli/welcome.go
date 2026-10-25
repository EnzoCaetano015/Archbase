package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// Each glyph has the same width so the wordmark stays aligned in terminal fonts.
var welcomeLetters = map[rune][]string{
	'A': {" █████ ", "██   ██", "██   ██", "███████", "██   ██", "██   ██"},
	'R': {"██████ ", "██   ██", "██   ██", "██████ ", "██  ██ ", "██   ██"},
	'C': {" ██████", "██     ", "██     ", "██     ", "██     ", " ██████"},
	'H': {"██   ██", "██   ██", "██   ██", "███████", "██   ██", "██   ██"},
	'B': {"██████ ", "██   ██", "██████ ", "██   ██", "██   ██", "██████ "},
	'S': {" ██████", "██     ", " █████ ", "     ██", "     ██", "██████ "},
	'E': {"███████", "██     ", "██████ ", "██     ", "██     ", "███████"},
}

func welcome(output io.Writer, version string) error {
	color := false
	if file, ok := output.(*os.File); ok && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb" {
		var restore func()
		color, restore = welcomeTerminal(file)
		defer restore()
	}
	return renderWelcome(output, version, color)
}

func renderWelcome(output io.Writer, version string, color bool) error {
	var text strings.Builder
	text.WriteByte('\n')
	// Bright highlights fade to graphite, giving the block letters a silver finish.
	shades := [...]int{255, 253, 250, 247, 244, 240}
	for row, shade := range shades {
		if color {
			fmt.Fprintf(&text, "\x1b[38;5;%dm", shade)
		}
		for column, letter := range "ARCHBASE" {
			if column > 0 {
				text.WriteByte(' ')
			}
			text.WriteString(welcomeLetters[letter][row])
		}
		if color {
			text.WriteString("\x1b[0m")
		}
		text.WriteByte('\n')
	}
	fmt.Fprintf(&text, "\n  Welcome to Archbase  /  %s\n", version)
	text.WriteString("  Architecture that AI coding agents can actually follow.\n\n")
	text.WriteString("  Get started\n")
	text.WriteString("    arc add next/page@1234 .    Install your first pattern\n")
	text.WriteString("    arc rules list             Explore architecture rules\n")
	text.WriteString("    arc update                 Install the latest stable CLI\n")
	text.WriteString("    arc --help                 See all commands and options\n\n")
	_, err := io.WriteString(output, text.String())
	return err
}
