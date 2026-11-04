package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/EnzoCaetano015/Archbase/internal/diagnostics"
	"github.com/EnzoCaetano015/Archbase/internal/registry"
)

func TestDoctorJSONIsOfflineAndStdoutIsOneObject(t *testing.T) {
	root := doctorTempDir(t)
	var stdout, stderr bytes.Buffer
	deps := Dependencies{ResolverFactory: func(context.Context, RegistryOptions) (*registry.Resolver, error) {
		t.Fatal("doctor initialized a network-capable resolver")
		return nil, errors.New("unexpected")
	}}
	code := Execute([]string{"--registry-url", "https://invalid.example/never.git", "doctor", "--json", "--project-root", root}, &stdout, &stderr, "9.8.7", deps)
	var result diagnostics.Result
	if code != 0 || json.Unmarshal(stdout.Bytes(), &result) != nil || stderr.Len() != 0 || result.ToolVersion != "9.8.7" {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if strings.Contains(stdout.String(), root) {
		t.Fatal("absolute root leaked")
	}
}

func TestDoctorExitCodesAndHumanOutput(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want int
	}{{"missing-root", []string{"doctor", "--json", "--project-root", filepath.Join(doctorTempDir(t), "missing")}, 2}, {"unknown-flag", []string{"doctor", "--invalid"}, 2}, {"argument", []string{"doctor", "unexpected"}, 2}, {"unconfigured", []string{"doctor", "--project-root", doctorTempDir(t)}, 0}} {
		t.Run(tc.name, func(t *testing.T) {
			var out, err bytes.Buffer
			code := Execute(tc.args, &out, &err, "dev")
			if code != tc.want {
				t.Fatalf("%d want %d: %s %s", code, tc.want, out.String(), err.String())
			}
			if tc.name == "missing-root" {
				var result diagnostics.Result
				if json.Unmarshal(out.Bytes(), &result) != nil || err.Len() != 0 {
					t.Fatal("invalid root must produce only the typed JSON result")
				}
			}
			if tc.name == "unconfigured" && !strings.Contains(out.String(), "Workspace is not configured") {
				t.Fatal(out.String())
			}
		})
	}
}

func TestDoctorMalformedWorkspaceDoesNotEchoInput(t *testing.T) {
	root := doctorTempDir(t)
	if err := os.Mkdir(filepath.Join(root, ".archbase"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".archbase/scopes.yaml"), []byte("secret: DO_NOT_ECHO\nscopes: ["), 0644); err != nil {
		t.Fatal(err)
	}
	var out, err bytes.Buffer
	code := Execute([]string{"doctor", "--json", "--project-root", root}, &out, &err, "dev")
	if code != 3 || err.Len() != 0 || strings.Contains(out.String(), "DO_NOT_ECHO") {
		t.Fatalf("%d %s %s", code, out.String(), err.String())
	}
	var result diagnostics.Result
	if json.Unmarshal(out.Bytes(), &result) != nil {
		t.Fatal("not JSON")
	}
}

func TestDoctorInvalidArgumentsStillProduceJSON(t *testing.T) {
	var out, stderr bytes.Buffer
	code := Execute([]string{"doctor", "--json", "must-not-be-echoed"}, &out, &stderr, "dev")
	var result diagnostics.Result
	if code != 2 || stderr.Len() != 0 || json.Unmarshal(out.Bytes(), &result) != nil || result.State != "error" || len(result.Checks) != 10 || strings.Contains(out.String(), "must-not-be-echoed") {
		t.Fatalf("%d %s %s", code, out.String(), stderr.String())
	}
}

func doctorTempDir(t *testing.T) string {
	t.Helper()
	p, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return p
}
