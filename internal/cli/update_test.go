package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/EnzoCaetano015/Archbase/internal/registry"
	"github.com/EnzoCaetano015/Archbase/internal/update"
)

func TestUpdateCommandUsesIsolatedRunner(t *testing.T) {
	for _, args := range [][]string{{"update"}, {"update", "--check", "--version", "v2.0.0", "--force"}} {
		called := false
		deps := Dependencies{UpdateRunner: func(ctx context.Context, version string, options update.Options) (update.Result, error) {
			called = true
			if version != "1.0.0" {
				t.Fatal(version)
			}
			if len(args) > 1 && (!options.Check || !options.Force || options.Version != "v2.0.0") {
				t.Fatalf("options: %#v", options)
			}
			return update.Result{Current: version, Latest: "2.0.0", Available: true, Updated: !options.Check, Warnings: []string{"preview warning"}}, nil
		}, ResolverFactory: func(context.Context, RegistryOptions) (*registry.Resolver, error) {
			t.Fatal("update must not load registry or workspace")
			return nil, nil
		}}
		var out, stderr bytes.Buffer
		if code := Execute(args, &out, &stderr, "1.0.0", deps); code != 0 || !called || !strings.Contains(stderr.String(), "preview warning") {
			t.Fatalf("code=%d out=%s err=%s", code, &out, &stderr)
		}
		if !strings.Contains(out.String(), "1.0.0 -> 2.0.0") {
			t.Fatal(out.String())
		}
	}
}

func TestUpdateCommandErrorsAndAlreadyCurrent(t *testing.T) {
	var out, stderr bytes.Buffer
	deps := Dependencies{UpdateRunner: func(context.Context, string, update.Options) (update.Result, error) {
		return update.Result{}, errors.New("checksum mismatch")
	}}
	if code := Execute([]string{"update"}, &out, &stderr, "1.0.0", deps); code != 1 || !strings.Contains(stderr.String(), "checksum mismatch") {
		t.Fatal(code, &stderr)
	}
	for _, args := range [][]string{{"update", "extra"}, {"update", "--unknown"}} {
		if code := Execute(args, &out, &stderr, "1.0.0", deps); code != 1 {
			t.Fatal("invalid arguments accepted")
		}
	}
	deps.UpdateRunner = func(context.Context, string, update.Options) (update.Result, error) {
		return update.Result{Current: "2.0.0", Latest: "2.0.0"}, nil
	}
	out.Reset()
	stderr.Reset()
	if code := Execute([]string{"update"}, &out, &stderr, "2.0.0", deps); code != 0 || !strings.Contains(out.String(), "up to date") {
		t.Fatal(code, &out, &stderr)
	}
}
