package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestHelpListsCurrentAndPlannedCommands(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"help"}, &stdout, &stderr, "dev")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d: %s", code, stderr.String())
	}
	for _, expected := range []string{"arc version", "arc add", "arc mcp serve"} {
		if !strings.Contains(stdout.String(), expected) {
			t.Errorf("help output does not contain %q", expected)
		}
	}
}

func TestVersion(t *testing.T) {
	for _, version := range []string{"dev", "1.2.3"} {
		t.Run(version, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Execute([]string{"version"}, &stdout, &stderr, version)
			expected := "arc " + version + "\n"
			if code != 0 || stdout.String() != expected {
				t.Fatalf("unexpected result: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
		})
	}
}

func TestUnknownCommandReturnsNonZero(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"add"}, &stdout, &stderr, "dev")
	if code == 0 {
		t.Fatal("expected a non-zero exit code")
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("unexpected error: %q", stderr.String())
	}
}
