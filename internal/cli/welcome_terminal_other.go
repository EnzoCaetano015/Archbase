//go:build !linux && !darwin && !windows

package cli

import "os"

func welcomeTerminal(_ *os.File) (bool, func()) {
	return false, func() {}
}
