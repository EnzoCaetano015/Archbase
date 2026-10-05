package cli

import (
	"os"

	"golang.org/x/sys/windows"
)

func welcomeTerminal(file *os.File) (bool, func()) {
	handle := windows.Handle(file.Fd())
	var mode uint32
	if windows.GetConsoleMode(handle, &mode) != nil {
		return false, func() {}
	}
	if mode&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING != 0 {
		return true, func() {}
	}
	if windows.SetConsoleMode(handle, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING) != nil {
		return false, func() {}
	}
	return true, func() { _ = windows.SetConsoleMode(handle, mode) }
}
