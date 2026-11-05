package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/EnzoCaetano015/Archbase/internal/diagnostics"
	"github.com/charmbracelet/x/ansi"
)

type progressCapture struct {
	sync.Mutex
	bytes.Buffer
	shown chan struct{}
	once  sync.Once
}

func (w *progressCapture) Write(p []byte) (int, error) {
	w.Lock()
	defer w.Unlock()
	n, err := w.Buffer.Write(p)
	w.once.Do(func() { close(w.shown) })
	return n, err
}

func TestProgressDelayAndCleanup(t *testing.T) {
	var fast bytes.Buffer
	stop := animateProgress(&fast, "Fast", 60, time.Hour)
	stop()
	stop()
	if fast.Len() != 0 {
		t.Fatal("fast operation animated")
	}
	w := &progressCapture{shown: make(chan struct{})}
	stop = animateProgress(w, "Working", 60, time.Millisecond)
	select {
	case <-w.shown:
	case <-time.After(3 * time.Second):
		stop()
		t.Fatal("spinner did not start")
	}
	stop()
	stop()
	if !strings.Contains(w.String(), "Working") || !strings.HasSuffix(w.String(), "\r\x1b[2K") {
		t.Fatalf("missing label or cleanup: %q", w.String())
	}
}

func TestDoctorPresentation(t *testing.T) {
	r := diagnostics.Result{ToolVersion: "dev", State: "partial", Checks: []diagnostics.Check{
		{ID: "doctor.cli-version", Status: "passed", Message: "CLI details"},
		{ID: "doctor.workspace-index", Status: "warning", Message: "Workspace is not configured.", Reason: "missing-workspace", Recommendation: "Review configuration"},
		{ID: "doctor.scope-paths", Status: "error", Message: "Unsafe scope"},
		{ID: "doctor.local-bundles", Status: "not-applicable", Message: "No local patterns"},
		{ID: "doctor.snapshot", Status: "not-run", Message: "Snapshot unavailable", Reason: "read-budget-exhausted"},
	}}
	compact := renderDoctor(r, false, newTheme(false), 120)
	for _, want := range []string{"[✓]", "[!]", "[✗]", "[-]", "[?]", "partial (incomplete)", "read budget exhausted", "1 passed", "1 warning", "1 error"} {
		if !strings.Contains(compact, want) {
			t.Errorf("missing %q: %s", want, compact)
		}
	}
	if strings.Contains(compact, "doctor.") || strings.Contains(compact, "Reason:") || strings.Contains(compact, "Review configuration") {
		t.Fatal(compact)
	}
	verbose := renderDoctor(r, true, newTheme(false), 120)
	for _, want := range []string{"doctor.workspace-index", "Reason: missing-workspace", "Next: Review configuration", "CLI details"} {
		if !strings.Contains(verbose, want) {
			t.Error(verbose)
		}
	}
	r.Checks[4].Reason = "cancelled"
	if !strings.Contains(renderDoctor(r, false, newTheme(false), 120), "cancelled (incomplete)") {
		t.Fatal("cancellation hidden")
	}
	r.Checks[1].Message = "\x1b]0;malicious\aSafe message"
	for _, width := range []int{60, 80, 120} {
		out := renderDoctor(r, true, newTheme(false), width)
		if strings.Contains(out, "\x1b") || strings.Contains(out, "malicious") {
			t.Fatal("unsafe terminal content")
		}
		for _, line := range strings.Split(out, "\n") {
			if ansi.StringWidth(line) > width {
				t.Fatal("line overflow")
			}
		}
	}
}

func TestDoctorPlainAndJSONCompatibility(t *testing.T) {
	root := doctorTempDir(t)
	for _, args := range [][]string{{"doctor", "--project-root", root}, {"doctor", "-v", "--project-root", root}, {"--plain", "doctor", "--project-root", root}} {
		var out, err bytes.Buffer
		if Execute(args, &out, &err, "dev") != 0 || err.Len() != 0 || !strings.Contains(out.String(), "[passed] doctor.cli-version") || strings.Contains(out.String(), "\x1b") {
			t.Fatalf("plain output changed: %s %s", &out, &err)
		}
	}
	var out, err bytes.Buffer
	if Execute([]string{"doctor", "--json", "-v", "--project-root", root}, &out, &err, "dev") != 0 || !json.Valid(out.Bytes()) || err.Len() != 0 {
		t.Fatal(&out, &err)
	}
}
