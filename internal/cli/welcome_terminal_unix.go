//go:build linux || darwin

package cli

import (
	"os"

	"golang.org/x/sys/unix"
)

func welcomeTerminal(file *os.File) (bool, func()) {
	_, err := unix.IoctlGetWinsize(int(file.Fd()), unix.TIOCGWINSZ)
	return err == nil, func() {}
}
