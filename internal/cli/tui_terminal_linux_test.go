//go:build linux

package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// Exercise the actual entry point in a PTY, including raw-mode restoration.
// Model-only tests cannot detect a broken TTY gate or leaked alternate screen.
func TestInteractiveTerminalLifecycle(t *testing.T) {
	for _, scenario := range []string{"navigation", "operation-failure"} {
		t.Run(scenario, func(t *testing.T) { terminalLifecycle(t, scenario) })
	}
}

func terminalLifecycle(t *testing.T, scenario string) {
	fd, err := unix.Open("/dev/ptmx", unix.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		t.Skipf("PTY unavailable: %v", err)
	}
	master := os.NewFile(uintptr(fd), "ptmx")
	defer master.Close()
	if err := unix.IoctlSetPointerInt(fd, unix.TIOCSPTLCK, 0); err != nil {
		t.Fatal(err)
	}
	number, err := unix.IoctlGetInt(fd, unix.TIOCGPTN)
	if err != nil {
		t.Fatal(err)
	}
	slave, err := os.OpenFile(fmt.Sprintf("/dev/pts/%d", number), os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer slave.Close()
	if err := unix.IoctlSetWinsize(fd, unix.TIOCSWINSZ, &unix.Winsize{Row: 32, Col: 100}); err != nil {
		t.Fatal(err)
	}
	before, err := unix.IoctlGetTermios(int(slave.Fd()), unix.TCGETS)
	if err != nil {
		t.Fatal(err)
	}
	executable, _ := os.Executable()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, executable, "-test.run=^TestInteractiveTerminalHelper$")
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, "TERM=") && !strings.HasPrefix(value, "NO_COLOR=") {
			command.Env = append(command.Env, value)
		}
	}
	command.Env = append(command.Env, "ARCHBASE_TUI_HELPER=1", "TERM=xterm-256color", "NO_COLOR=1")
	command.Stdin, command.Stdout, command.Stderr = slave, slave, slave
	command.Dir = t.TempDir()
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var output bytes.Buffer
	go func() {
		buffer := make([]byte, 4096)
		for {
			n, err := master.Read(buffer)
			mu.Lock()
			output.Write(buffer[:n])
			mu.Unlock()
			if err != nil {
				return
			}
		}
	}()
	waitFor := func(want string) {
		t.Helper()
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			mu.Lock()
			found := bytes.Contains(output.Bytes(), []byte(want))
			mu.Unlock()
			if found {
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		cancel()
		_ = command.Wait()
		mu.Lock()
		defer mu.Unlock()
		t.Fatalf("terminal did not display %q: %q", want, output.String())
	}
	waitFor("Home")
	if scenario == "operation-failure" {
		if _, err := io.WriteString(master, "\x10resolve\r"); err != nil {
			t.Fatal(err)
		}
		waitFor("Target path")
		if _, err := io.WriteString(master, "\x13"); err != nil {
			t.Fatal(err)
		}
		waitFor("Operation failed")
	} else {
		if _, err := io.WriteString(master, "\x10create\r"); err != nil {
			t.Fatal(err)
		}
		waitFor("Local name")
	}
	if _, err := io.WriteString(master, "\x1b"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	if _, err := io.WriteString(master, "\x03"); err != nil {
		t.Fatal(err)
	}
	if err := command.Wait(); err != nil {
		t.Fatal(err)
	}
	waitFor("\x1b[?1049l")
	after, err := unix.IoctlGetTermios(int(slave.Fd()), unix.TCGETS)
	if err != nil {
		t.Fatal(err)
	}
	if *before != *after {
		t.Fatal("terminal mode was not restored")
	}
}

func TestInteractiveTerminalHelper(t *testing.T) {
	if os.Getenv("ARCHBASE_TUI_HELPER") != "1" {
		return
	}
	os.Exit(Execute(nil, os.Stdout, os.Stderr, "test"))
}
