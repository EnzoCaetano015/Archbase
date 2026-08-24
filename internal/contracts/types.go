package contracts

type Metadata map[string]any

type Manifest struct {
	SchemaVersion  int              `yaml:"schemaVersion" json:"schemaVersion"`
	ID             string           `yaml:"id" json:"id"`
	Name           string           `yaml:"name" json:"name"`
	Description    string           `yaml:"description,omitempty" json:"description,omitempty"`
	Type           string           `yaml:"type" json:"type"`
	Version        string           `yaml:"version" json:"version"`
	Structure      PatternStructure `yaml:"structure" json:"structure"`
	AllowedChanges []string         `yaml:"allowedChanges" json:"allowedChanges"`
	Preserve       []string         `yaml:"preserve" json:"preserve"`
	Metadata       Metadata         `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

type PatternStructure struct {
	Root  string        `yaml:"root" json:"root"`
	Files []PatternFile `yaml:"files" json:"files"`
}

type PatternFile struct {
	Source      string `yaml:"source" json:"source"`
	Destination string `yaml:"destination" json:"destination"`
	Required    bool   `yaml:"required" json:"required"`
}

type Scope struct {
	SchemaVersion int           `yaml:"schemaVersion" json:"schemaVersion"`
	Scope         ScopeSelector `yaml:"scope" json:"scope"`
	Pattern       ScopePattern  `yaml:"pattern" json:"pattern"`
	Origin        *Origin       `yaml:"origin,omitempty" json:"origin,omitempty"`
	Behavior      ScopeBehavior `yaml:"behavior" json:"behavior"`
	Metadata      Metadata      `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

type ScopeSelector struct {
	Path string `yaml:"path" json:"path"`
}

type ScopePattern struct {
	ID     string `yaml:"id" json:"id"`
	Source string `yaml:"source" json:"source"`
	Root   string `yaml:"root,omitempty" json:"root,omitempty"`
}

type Origin struct {
	Registry string `yaml:"registry" json:"registry"`
	ID       string `yaml:"id" json:"id"`
	Version  string `yaml:"version" json:"version"`
}

type ScopeBehavior struct {
	NearestScopeWins        bool `yaml:"nearestScopeWins" json:"nearestScopeWins"`
	AllowLocalCustomization bool `yaml:"allowLocalCustomization" json:"allowLocalCustomization"`
}

type Rule struct {
	SchemaVersion int         `yaml:"schemaVersion" json:"schemaVersion"`
	ID            string      `yaml:"id" json:"id"`
	Name          string      `yaml:"name" json:"name"`
	Description   string      `yaml:"description,omitempty" json:"description,omitempty"`
	Version       string      `yaml:"version" json:"version"`
	Scopes        []RuleScope `yaml:"scopes" json:"scopes"`
	Rules         []string    `yaml:"rules" json:"rules"`
	Metadata      Metadata    `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

type RuleScope struct {
	Path    string `yaml:"path" json:"path"`
	Pattern string `yaml:"pattern" json:"pattern"`
}
