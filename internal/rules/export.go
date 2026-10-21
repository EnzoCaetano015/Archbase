package rules

import (
	"fmt"
	"strings"

	"github.com/EnzoCaetano015/Archbase/internal/contracts"
)

type Format string

const (
	FormatCursor  Format = "cursor"
	FormatCopilot Format = "copilot"
	FormatAgents  Format = "agents"
	FormatClaude  Format = "claude"
)

func ParseFormat(value string) (Format, error) {
	format := Format(value)
	switch format {
	case FormatCursor, FormatCopilot, FormatAgents, FormatClaude:
		return format, nil
	default:
		return "", fmt.Errorf("unsupported rules format %q; expected agents, claude, cursor, or copilot", value)
	}
}

type Artifact struct {
	RelativePath string
	Content      []byte
	StartMarker  string
	EndMarker    string
}

func Render(rule Rule, format Format) ([]Artifact, error) {
	if _, err := ParseFormat(string(format)); err != nil {
		return nil, err
	}
	document, err := renderDocument(rule)
	if err != nil {
		return nil, err
	}
	m, err := readDocument(document.RelativePath, document.Content)
	if err != nil {
		return nil, err
	}
	block := renderIndex(format, map[string]documentMetadata{m.ID: m})
	return []Artifact{document, {RelativePath: entryPaths[format], Content: newEntry(format, block), StartMarker: indexStart, EndMarker: indexEnd}}, nil
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

func managedMarkers(id string) (string, string) {
	return "<!-- archbase:rule " + id + " start -->", "<!-- archbase:rule " + id + " end -->"
}

func ownershipMarker(id string) string {
	return "<!-- archbase:rule " + id + " export -->"
}

func ruleSlug(id string) string {
	replacer := strings.NewReplacer("/", "-", "@", "-")
	return replacer.Replace(id)
}
