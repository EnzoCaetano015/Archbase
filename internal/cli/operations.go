package cli

import (
	"context"
	"fmt"

	"github.com/EnzoCaetano015/Archbase/internal/registry"
	archrules "github.com/EnzoCaetano015/Archbase/internal/rules"
	"github.com/EnzoCaetano015/Archbase/internal/workspace"
)

type action string

const (
	listPatterns   action = "patterns"
	listRules      action = "rules"
	inspectPattern action = "inspect"
	inspectRule    action = "rules inspect"
	resolvePath    action = "resolve"
	addPattern     action = "add"
	createPattern  action = "create"
	exportRule     action = "rules add"
	removeRule     action = "remove"
)

type request struct {
	Action                 action
	ID, Path, From, Format string
	Overwrite, Merge, All  bool
	WithNames              bool
}

func (r request) writes() bool {
	return r.Action == addPattern || r.Action == createPattern || r.Action == exportRule || r.Action == removeRule
}

// operationResult is shared by the shell and the interactive application.
// Neither presentation parses text or launches the CLI as a subprocess.
type operationResult struct {
	Action     action
	Patterns   []registry.Entry
	Rules      []archrules.Entry
	Pattern    *registry.Pattern
	Rule       *archrules.Rule
	Resolution *workspace.Resolution
	Installed  *workspace.Installed
	Exported   *archrules.ExportResult
	Removed    *archrules.RemoveResult
	ID         string
	Warnings   []error
	Names      map[string]string
}

type operations struct {
	dependencies Dependencies
	options      RegistryOptions
}

func (o operations) run(ctx context.Context, r request) (operationResult, error) {
	result := operationResult{Action: r.Action, ID: r.ID}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	// Removal needs only local export ownership, never a registry connection.
	if r.Action == removeRule {
		exporter, err := archrules.NewExporter(o.dependencies.RuleFileSystem)
		if err != nil {
			return result, err
		}
		removed, err := exporter.Remove(r.ID, archrules.RemoveOptions{Destination: r.Path, All: r.All})
		result.Removed = &removed
		return result, err
	}
	if r.Action == exportRule {
		if _, err := archrules.ParseFormat(r.Format); err != nil {
			return result, err
		}
	}
	ws, patterns, err := service(ctx, o.dependencies, o.options)
	if err != nil {
		return result, err
	}
	switch r.Action {
	case listPatterns:
		listed, err := patterns.List(ctx)
		result.Patterns, result.Warnings = listed.Entries, listed.Warnings
		if err == nil && r.WithNames {
			result.Names = make(map[string]string)
			for _, entry := range listed.Entries {
				if err := ctx.Err(); err != nil {
					return result, err
				}
				resolved, err := patterns.Resolve(ctx, entry.ID.String())
				if err != nil {
					return result, err
				}
				result.Names[entry.ID.String()] = resolved.Pattern.Bundle.Manifest.Name
			}
		}
		return result, err
	case inspectPattern:
		if _, parseErr := registry.ParsePatternID(r.ID); parseErr == nil {
			resolved, err := patterns.Resolve(ctx, r.ID)
			result.Pattern, result.Warnings = &resolved.Pattern, resolved.Warnings
			return result, err
		}
		resolved, err := ws.Resolve(ctx, r.ID)
		result.Resolution, result.Pattern, result.Warnings = &resolved, &resolved.Pattern, resolved.Warnings
		return result, err
	case resolvePath:
		resolved, err := ws.Resolve(ctx, r.Path)
		result.Resolution, result.Warnings = &resolved, resolved.Warnings
		return result, err
	case addPattern, createPattern:
		var installed workspace.Installed
		if r.Action == addPattern {
			installed, err = ws.Add(ctx, r.ID, r.Path)
		} else {
			installed, err = ws.Create(ctx, r.ID, r.Path, r.From)
		}
		result.Installed, result.Warnings = &installed, installed.Warnings
		return result, err
	case listRules, inspectRule, exportRule:
		rules, err := o.dependencies.RuleResolverFactory(ctx, o.options, patterns)
		if err != nil {
			return result, err
		}
		if r.Action == listRules {
			listed, err := rules.List(ctx)
			result.Rules, result.Warnings = listed.Entries, listed.Warnings
			if err == nil && r.WithNames {
				result.Names = make(map[string]string)
				for _, entry := range listed.Entries {
					if err := ctx.Err(); err != nil {
						return result, err
					}
					resolved, err := rules.Resolve(ctx, entry.ID.String())
					if err != nil {
						return result, err
					}
					result.Names[entry.ID.String()] = resolved.Rule.Definition.Name
				}
			}
			return result, err
		}
		resolved, err := rules.Resolve(ctx, r.ID)
		if err != nil {
			return result, err
		}
		result.Rule, result.Warnings = &resolved.Rule, resolved.Warnings
		if r.Action == inspectRule {
			return result, nil
		}
		format, _ := archrules.ParseFormat(r.Format)
		exporter, err := archrules.NewExporter(o.dependencies.RuleFileSystem)
		if err != nil {
			return result, err
		}
		exported, err := exporter.Export(resolved.Rule, format, archrules.ExportOptions{Destination: r.Path, Overwrite: r.Overwrite, Merge: r.Merge})
		result.Exported = &exported
		return result, err
	default:
		return result, fmt.Errorf("unknown action %q", r.Action)
	}
}
