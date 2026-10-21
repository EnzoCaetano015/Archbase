package rules

import (
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
)

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
	output.WriteString(ownershipMarker(rule.Definition.ID))
	output.WriteString("\n\n")
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
	output.WriteString(ownershipMarker(rule.Definition.ID))
	output.WriteString("\n\n")
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
