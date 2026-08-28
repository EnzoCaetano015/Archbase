package mcp

import "github.com/EnzoCaetano015/Archbase/internal/contracts"

type SearchPatternsInput struct {
	Query string `json:"query,omitempty" jsonschema:"optional case-insensitive text matched against pattern ID, name, and description"`
}

type PatternIDInput struct {
	PatternID string `json:"patternId" jsonschema:"canonical pattern ID in stack/type@id form"`
}

type PathInput struct {
	Path string `json:"path" jsonschema:"project-relative path or absolute path confined to the configured project root"`
}

type PatternFilesInput struct {
	PatternID string `json:"patternId,omitempty" jsonschema:"canonical registry pattern ID; mutually exclusive with path"`
	Path      string `json:"path,omitempty" jsonschema:"project path resolved through the nearest scope; mutually exclusive with patternId"`
}

type EmptyInput struct{}

type PatternSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Source      string `json:"source"`
}

type SearchPatternsOutput struct {
	Patterns []PatternSummary `json:"patterns"`
	Stale    bool             `json:"stale"`
	Warnings []string         `json:"warnings"`
}

type PatternOutput struct {
	Manifest contracts.Manifest `json:"manifest"`
	Source   string             `json:"source"`
	Stale    bool               `json:"stale"`
	Warnings []string           `json:"warnings"`
}

type FileMetadata struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Required    bool   `json:"required"`
	Present     bool   `json:"present"`
}

type ResolvedPatternOutput struct {
	ScopePath   string             `json:"scopePath"`
	PatternRoot string             `json:"patternRoot,omitempty"`
	Manifest    contracts.Manifest `json:"manifest"`
	Source      string             `json:"source"`
	Origin      *contracts.Origin  `json:"origin,omitempty"`
	Files       []FileMetadata     `json:"files"`
	Stale       bool               `json:"stale"`
	Warnings    []string           `json:"warnings"`
}

type PatternFileContent struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Required    bool   `json:"required"`
	Present     bool   `json:"present"`
	Encoding    string `json:"encoding"`
	Content     string `json:"content"`
}

type PatternFilesOutput struct {
	PatternID string               `json:"patternId"`
	Source    string               `json:"source"`
	Files     []PatternFileContent `json:"files"`
	Stale     bool                 `json:"stale"`
	Warnings  []string             `json:"warnings"`
}

type RuleMatch struct {
	ID             string                `json:"id"`
	Name           string                `json:"name"`
	Version        string                `json:"version"`
	Description    string                `json:"description"`
	Source         string                `json:"source"`
	MatchingScopes []contracts.RuleScope `json:"matchingScopes"`
	Restrictions   []string              `json:"restrictions"`
}

type ScopeRulesOutput struct {
	PatternID string      `json:"patternId"`
	OriginID  string      `json:"originId,omitempty"`
	Rules     []RuleMatch `json:"rules"`
	Stale     bool        `json:"stale"`
	Warnings  []string    `json:"warnings"`
}

type ProjectScope struct {
	Path    string                 `json:"path"`
	Pattern contracts.ScopePattern `json:"pattern"`
	Origin  *contracts.Origin      `json:"origin,omitempty"`
}

type ProjectScopesOutput struct {
	Scopes   []ProjectScope `json:"scopes"`
	Stale    bool           `json:"stale"`
	Warnings []string       `json:"warnings"`
}
