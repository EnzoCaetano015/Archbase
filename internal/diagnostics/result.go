// Package diagnostics implements offline, read-only project diagnostics.
package diagnostics

import "sort"

type Check struct {
	ID             string  `json:"id"`
	Status         string  `json:"status"`
	Path           *string `json:"path"`
	Message        string  `json:"message"`
	Reason         string  `json:"reason,omitempty"`
	Recommendation string  `json:"recommendation,omitempty"`
}

type Result struct {
	SchemaVersion int     `json:"schemaVersion"`
	Command       string  `json:"command"`
	ToolVersion   string  `json:"toolVersion"`
	ProjectRoot   string  `json:"projectRoot"`
	State         string  `json:"state"`
	Checks        []Check `json:"checks"`
}

// InvalidInvocation preserves JSON output without inspecting a project when
// unsupported positional arguments are supplied.
func InvalidInvocation(version string) Result {
	if version == "" {
		version = "dev"
	}
	r := Result{SchemaVersion: 1, Command: "doctor", ToolVersion: version, ProjectRoot: ".", State: "error", Checks: []Check{}}
	for _, id := range checkIDs {
		c := Check{ID: "doctor." + id, Status: "not-run", Message: "Diagnostic did not start because the invocation is invalid.", Reason: "invalid-invocation"}
		if id == "cli-version" {
			c.Status = "passed"
			c.Message = "Invoked CLI version reported without a network lookup."
			c.Reason = ""
		}
		if id == "project-root" {
			c.Status = "error"
			c.Message = "Doctor accepts no positional arguments; select a root with --project-root."
		}
		r.Checks = append(r.Checks, c)
	}
	r.finish()
	return r
}

// ExitCode is shared by human and JSON output. Cancellation takes precedence.
func (r Result) ExitCode() int {
	for _, c := range r.Checks {
		if c.Reason == "cancelled" {
			return 130
		}
	}
	if r.State == "error" {
		return 2
	}
	if r.State == "partial" {
		return 3
	}
	for _, c := range r.Checks {
		if c.Status == "error" {
			return 1
		}
	}
	return 0
}

func (r *Result) finish() {
	if r.State != "error" {
		r.State = "complete"
		for _, c := range r.Checks {
			if c.Status == "not-run" {
				r.State = "partial"
				break
			}
		}
	}
	sort.Slice(r.Checks, func(i, j int) bool { return r.Checks[i].ID < r.Checks[j].ID })
}
