package rules

import (
	"bytes"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/EnzoCaetano015/Archbase/internal/contracts"
	"github.com/goccy/go-yaml"
)

const indexStart = "<!-- archbase:rules start -->"
const indexEnd = "<!-- archbase:rules end -->"

var formats = []Format{FormatAgents, FormatClaude, FormatCopilot, FormatCursor}
var entryPaths = map[Format]string{FormatAgents: "AGENTS.md", FormatClaude: "CLAUDE.md", FormatCopilot: ".github/copilot-instructions.md", FormatCursor: ".cursor/rules/archbase.mdc"}

type documentMetadata struct {
	ID          string                `yaml:"id"`
	Name        string                `yaml:"name"`
	Description string                `yaml:"description"`
	Version     string                `yaml:"version"`
	Source      string                `yaml:"source"`
	Scopes      []contracts.RuleScope `yaml:"scopes"`
}

func documentPath(id string) string { return path.Join(".archbase", "rules", ruleSlug(id)+".md") }

func renderDocument(rule Rule) (Artifact, error) {
	if _, err := contracts.EncodeRule(rule.Definition); err != nil {
		return Artifact{}, err
	}
	m := documentMetadata{ID: rule.Definition.ID, Name: rule.Definition.Name, Description: rule.Definition.Description, Version: rule.Definition.Version, Source: rule.Source, Scopes: rule.Definition.Scopes}
	if m.Source == "" {
		m.Source = rule.Entry.Source
	}
	if m.Source == "" {
		m.Source = "registry"
	}
	header, err := yaml.Marshal(m)
	if err != nil {
		return Artifact{}, err
	}
	content := append([]byte("---\n"), header...)
	content = append(content, []byte("---\n\n"+ownershipMarker(m.ID)+"\n\n"+renderRuleBody(rule, scopeBindings(m.Scopes)))...)
	return Artifact{RelativePath: documentPath(m.ID), Content: content}, nil
}

func frontmatter(content []byte) ([]byte, error) {
	text := strings.ReplaceAll(string(content), "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return nil, errors.New("missing YAML frontmatter")
	}
	end := strings.Index(text[4:], "\n---\n")
	if end < 0 && strings.HasSuffix(text[4:], "\n---") {
		end = len(text[4:]) - len("\n---")
	}
	if end < 0 {
		return nil, errors.New("unclosed YAML frontmatter")
	}
	return []byte(text[4 : 4+end]), nil
}

func readDocument(relative string, content []byte) (documentMetadata, error) {
	var m documentMetadata
	if bytes.Contains(content, []byte("<!-- archbase:rules")) {
		return m, errors.New("index markers are invalid in a central rule document")
	}
	id, owned, err := ownedExportRuleID(content)
	if err != nil {
		return m, err
	}
	if !owned {
		return m, errors.New("document is not owned by Archbase")
	}
	header, err := frontmatter(content)
	if err != nil {
		return m, err
	}
	if err = yaml.UnmarshalWithOptions(header, &m, yaml.Strict(), yaml.DisallowUnknownField()); err != nil {
		return m, err
	}
	if id != m.ID || relative != documentPath(m.ID) {
		return m, errors.New("document filename, metadata and ownership ID do not match")
	}
	if strings.TrimSpace(m.Source) == "" {
		return m, errors.New("document source is required")
	}
	// Reuse the canonical identity, SemVer and scope validators without parsing prose.
	_, err = contracts.EncodeRule(contracts.Rule{SchemaVersion: 1, ID: m.ID, Name: m.Name, Description: m.Description, Version: m.Version, Scopes: m.Scopes, Rules: []string{"Document body is Markdown."}})
	return m, err
}

func escapeMarkdown(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	return strings.NewReplacer("\\", "\\\\", "`", "\\`", "*", "\\*", "_", "\\_", "[", "\\[", "]", "\\]", "<", "&lt;", ">", "&gt;", "|", "\\|").Replace(value)
}

func renderIndex(format Format, documents map[string]documentMetadata) []byte {
	if len(documents) == 0 {
		return nil
	}
	ids := make([]string, 0, len(documents))
	for id := range documents {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	prefix := ""
	if format == FormatCopilot {
		prefix = "../"
	}
	if format == FormatCursor {
		prefix = "../../"
	}
	var b strings.Builder
	b.WriteString(indexStart + "\n\n## Archbase architecture rules\n\nBefore changing files, open and follow every linked rule whose scopes match the target paths. Scope paths are relative to the project root. These links are instructions to read the documents, not automatic imports.\n\n")
	for _, id := range ids {
		m := documents[id]
		description := m.Description
		if strings.TrimSpace(description) == "" {
			description = m.Name
		}
		scopes := make([]string, 0, len(m.Scopes))
		seen := map[string]bool{}
		for _, s := range m.Scopes {
			if !seen[s.Path] {
				scopes = append(scopes, escapeMarkdown(s.Path))
				seen[s.Path] = true
			}
		}
		fmt.Fprintf(&b, "- %s — %s — scopes: %s — [rule](%s%s)\n", escapeMarkdown(id), escapeMarkdown(description), strings.Join(scopes, ", "), prefix, documentPath(id))
	}
	b.WriteString("\n" + indexEnd + "\n")
	return []byte(b.String())
}

// indexRange also validates legacy block nesting, so one marker family cannot
// swallow another family's managed or external content during cleanup.
func indexRange(content []byte) (int, int, bool, error) {
	offset, start, end := 0, -1, -1
	activeLegacy := ""
	seen := map[string]bool{}
	for _, line := range splitLines(content) {
		text := trimLineEnding(line)
		switch {
		case strings.Contains(text, "<!-- archbase:rules"):
			if text == indexStart && start < 0 && end < 0 && activeLegacy == "" {
				start = offset
			} else if text == indexEnd && start >= 0 && end < 0 && activeLegacy == "" {
				end = offset + len(line)
			} else {
				return 0, 0, false, errors.New("managed index markers are incomplete, duplicated, reversed or overlapping")
			}
		case strings.Contains(text, "<!-- archbase:rule "):
			if start >= 0 && end < 0 {
				return 0, 0, false, errors.New("managed rule block overlaps the index")
			}
			id, kind, err := parseRuleMarker(text)
			if err != nil {
				return 0, 0, false, err
			}
			if kind == "start" && activeLegacy == "" && !seen[id] {
				activeLegacy = id
				seen[id] = true
			} else if kind == "end" && activeLegacy == id {
				activeLegacy = ""
			} else {
				return 0, 0, false, errors.New("legacy rule markers are incomplete, duplicated or overlapping")
			}
		}
		offset += len(line)
	}
	if (start >= 0 && end < 0) || activeLegacy != "" {
		return 0, 0, false, errors.New("managed markers are incomplete")
	}
	return start, end, start >= 0, nil
}

func replaceIndex(existing, replacement []byte) ([]byte, error) {
	start, end, found, err := indexRange(existing)
	if err != nil {
		return nil, err
	}
	if found {
		return bytes.Join([][]byte{existing[:start], replacement, existing[end:]}, nil), nil
	}
	if len(replacement) == 0 {
		return existing, nil
	}
	separator := ""
	if len(existing) > 0 {
		separator = "\n"
		if !bytes.HasSuffix(existing, []byte("\n")) {
			separator = "\n\n"
		}
	}
	return append(append(append([]byte(nil), existing...), []byte(separator)...), replacement...), nil
}

func validateCursor(content []byte) error {
	header, err := frontmatter(content)
	if err != nil {
		return err
	}
	var fields map[string]any
	if err = yaml.UnmarshalWithOptions(header, &fields, yaml.Strict()); err != nil {
		return err
	}
	if fields["alwaysApply"] != true {
		return errors.New("Cursor index requires alwaysApply: true; external frontmatter was preserved")
	}
	return nil
}

func newEntry(format Format, block []byte) []byte {
	if format == FormatCursor {
		return append([]byte("---\ndescription: Archbase architecture rule index\nalwaysApply: true\n---\n\n"), block...)
	}
	return block
}
