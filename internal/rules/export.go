package rules

import (
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/EnzoCaetano015/Archbase/internal/contracts"
)

type Format string

const (
	FormatCursor  Format = "cursor"
	FormatCopilot Format = "copilot"
	FormatAgents  Format = "agents"
)

func ParseFormat(value string) (Format, error) {
	format := Format(value)
	switch format {
	case FormatCursor, FormatCopilot, FormatAgents:
		return format, nil
	default:
		return "", fmt.Errorf("unsupported rules format %q; expected cursor, copilot, or agents", value)
	}
}

type Artifact struct {
	RelativePath string
	Content      []byte
	StartMarker  string
	EndMarker    string
}

func Render(rule Rule, format Format) ([]Artifact, error) {
	if _, err := ParseRuleID(rule.Definition.ID); err != nil {
		return nil, err
	}
	switch format {
	case FormatCursor:
		return renderCursor(rule), nil
	case FormatCopilot:
		return renderCopilot(rule)
	case FormatAgents:
		return renderAgents(rule), nil
	default:
		return nil, fmt.Errorf("unsupported rules format %q", format)
	}
}

func renderCursor(rule Rule) []Artifact {
	scopes := scopeBindings(rule.Definition.Scopes)
	var output strings.Builder
	output.WriteString("---\n")
	output.WriteString("description: ")
	output.WriteString(strconv.Quote(rule.Definition.Description))
	output.WriteString("\nglobs:\n")
	for _, scope := range rule.Definition.Scopes {
		output.WriteString("  - ")
		output.WriteString(strconv.Quote(scope.Path))
		output.WriteByte('\n')
	}
	output.WriteString("alwaysApply: false\n---\n\n")
	output.WriteString(renderRuleBody(rule, scopes))
	return []Artifact{{RelativePath: path.Join(".cursor", "rules", ruleSlug(rule.Definition.ID)+".mdc"), Content: []byte(output.String())}}
}

func renderCopilot(rule Rule) ([]Artifact, error) {
	scopes := scopeBindings(rule.Definition.Scopes)
	globs := make([]string, 0, len(rule.Definition.Scopes))
	for _, scope := range rule.Definition.Scopes {
		if strings.Contains(scope.Path, ",") {
			return nil, fmt.Errorf("Copilot export cannot represent a scope containing a comma: %q", scope.Path)
		}
		globs = append(globs, scope.Path)
	}
	var output strings.Builder
	output.WriteString("---\napplyTo: ")
	output.WriteString(strconv.Quote(strings.Join(globs, ",")))
	output.WriteString("\n---\n\n")
	output.WriteString(renderRuleBody(rule, scopes))
	return []Artifact{{RelativePath: path.Join(".github", "instructions", ruleSlug(rule.Definition.ID)+".instructions.md"), Content: []byte(output.String())}}, nil
}

func renderAgents(rule Rule) []Artifact {
	groups := make(map[string][]ScopeBinding)
	for _, scope := range rule.Definition.Scopes {
		base := staticScopeBase(scope.Path)
		groups[base] = append(groups[base], ScopeBinding{Path: scope.Path, Pattern: scope.Pattern})
	}
	bases := make([]string, 0, len(groups))
	for base := range groups {
		bases = append(bases, base)
	}
	sort.Strings(bases)
	artifacts := make([]Artifact, 0, len(bases))
	for _, base := range bases {
		start, end := managedMarkers(rule.Definition.ID)
		content := start + "\n" + renderRuleBody(rule, groups[base]) + end + "\n"
		relative := "AGENTS.md"
		if base != "." {
			relative = path.Join(base, relative)
		}
		artifacts = append(artifacts, Artifact{RelativePath: relative, Content: []byte(content), StartMarker: start, EndMarker: end})
	}
	return artifacts
}

// ScopeBinding is the exporter-facing path-to-pattern association.
type ScopeBinding struct {
	Path    string
	Pattern string
}

func scopeBindings(scopes []contracts.RuleScope) []ScopeBinding {
	bindings := make([]ScopeBinding, 0, len(scopes))
	for _, scope := range scopes {
		bindings = append(bindings, ScopeBinding{Path: scope.Path, Pattern: scope.Pattern})
	}
	return bindings
}

func renderRuleBody(rule Rule, scopes []ScopeBinding) string {
	var output strings.Builder
	fmt.Fprintf(&output, "# %s\n\n", rule.Definition.Name)
	if rule.Definition.Description != "" {
		fmt.Fprintf(&output, "%s\n\n", rule.Definition.Description)
	}
	output.WriteString("## Applicable scopes\n\n")
	for _, scope := range scopes {
		fmt.Fprintf(&output, "- `%s` uses pattern `%s`.\n", scope.Path, scope.Pattern)
	}
	output.WriteString("\n## Architecture rules\n\n")
	for _, restriction := range rule.Definition.Rules {
		fmt.Fprintf(&output, "- %s\n", restriction)
	}
	output.WriteString("\n## Pattern resolution\n\n")
	output.WriteString("Use `arc resolve <target-path>` to find the nearest active local pattern. ")
	output.WriteString("Use `arc inspect <pattern-id>` to inspect a referenced pattern before changing code.\n\n")
	return output.String()
}

func staticScopeBase(scope string) string {
	parts := strings.Split(scope, "/")
	static := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.ContainsAny(part, "*?[") {
			break
		}
		static = append(static, part)
	}
	if len(static) == 0 {
		return "."
	}
	return path.Join(static...)
}

func managedMarkers(id string) (string, string) {
	return "<!-- archbase:rule " + id + " start -->", "<!-- archbase:rule " + id + " end -->"
}

func ruleSlug(id string) string {
	replacer := strings.NewReplacer("/", "-", "@", "-")
	return replacer.Replace(id)
}
