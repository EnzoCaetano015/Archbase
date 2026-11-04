//go:build !windows

package diagnostics

import "syscall"

// A concurrently substituted FIFO must never block the read-only inspector.
const nonblockFlag = syscall.O_NONBLOCK
