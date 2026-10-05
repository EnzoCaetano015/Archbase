package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBareCommandWelcomesWithoutANSI(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Execute(nil, &stdout, &stderr, "1.2.3"); code != 0 {
		t.Fatalf("arc failed: code=%d stderr=%s", code, &stderr)
	}
	for _, want := range []string{"███████", "Welcome to Archbase  /  1.2.3", "arc add next/page@1234 .", "arc rules list", "arc --help"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("welcome is missing %q", want)
		}
	}
	if strings.Contains(stdout.String(), "\x1b[") || stderr.Len() != 0 {
		t.Fatalf("unexpected redirected output: stdout=%q stderr=%q", &stdout, &stderr)
	}
}

func TestExplicitHelpDoesNotIncludeWelcome(t *testing.T) {
	for _, args := range [][]string{{"help"}, {"--help"}, {"rules", "--help"}} {
		var stdout, stderr bytes.Buffer
		if code := Execute(args, &stdout, &stderr, "dev"); code != 0 {
			t.Fatalf("arc %v failed: %s", args, &stderr)
		}
		if strings.Contains(stdout.String(), "Welcome to Archbase") || strings.Contains(stdout.String(), "█") {
			t.Fatalf("arc %v includes the welcome screen", args)
		}
		if !strings.Contains(stdout.String(), "Usage:") {
			t.Fatalf("arc %v is missing usage", args)
		}
	}
}

func TestSilverWordmarkResetsEveryRow(t *testing.T) {
	var output bytes.Buffer
	if err := renderWelcome(&output, "dev", true); err != nil {
		t.Fatal(err)
	}
	if strings.Count(output.String(), "\x1b[0m") != 6 {
		t.Fatal("each banner row must reset its color")
	}
	for _, shade := range []string{"255", "253", "250", "247", "244", "240"} {
		if !strings.Contains(output.String(), "\x1b[38;5;"+shade+"m") {
			t.Errorf("missing silver shade %s", shade)
		}
	}
}

func TestWelcomeToFileHasNoANSI(t *testing.T) {
	for _, env := range []struct{ name, value string }{{"NO_COLOR", ""}, {"NO_COLOR", "1"}, {"TERM", "dumb"}} {
		t.Run(env.name+"="+env.value, func(t *testing.T) {
			t.Setenv(env.name, env.value)
			path := filepath.Join(t.TempDir(), "welcome.txt")
			file, err := os.Create(path)
			if err != nil {
				t.Fatal(err)
			}
			err = welcome(file, "dev")
			closeErr := file.Close()
			if err != nil || closeErr != nil {
				t.Fatalf("writing welcome: %v, close: %v", err, closeErr)
			}
			data, err := os.ReadFile(path)
			if err != nil || bytes.Contains(data, []byte("\x1b[")) {
				t.Fatalf("redirected output contains ANSI: %q, err=%v", data, err)
			}
		})
	}
}

type failedWelcomeWriter struct{ err error }

func (writer failedWelcomeWriter) Write([]byte) (int, error) { return 0, writer.err }

func TestWelcomeReportsWriteFailure(t *testing.T) {
	want := errors.New("output unavailable")
	var stderr bytes.Buffer
	if code := Execute(nil, failedWelcomeWriter{want}, &stderr, "dev"); code != 1 || !strings.Contains(stderr.String(), want.Error()) {
		t.Fatalf("expected a write failure: code=%d stderr=%q", code, &stderr)
	}
}
