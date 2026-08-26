package rules

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/EnzoCaetano015/Archbase/internal/contracts"
	"github.com/EnzoCaetano015/Archbase/internal/registry"
)

var (
	ErrRuleNotFound = errors.New("rule not found")
	ruleIDPattern   = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*/[a-z0-9][a-z0-9._-]*@[a-z0-9][a-z0-9._-]*$`)
)

type RuleID string

func ParseRuleID(value string) (RuleID, error) {
	if !ruleIDPattern.MatchString(value) {
		return "", fmt.Errorf("invalid rule ID %q; expected lowercase namespace/name@id", value)
	}
	return RuleID(value), nil
}

func (id RuleID) String() string { return string(id) }

type Entry struct {
	ID          RuleID
	Version     string
	Path        string
	Description string
	Source      string
}

type Rule struct {
	Entry      Entry
	Definition contracts.Rule
	Content    []byte
	Source     string
}

type LookupResult struct {
	Rule    Rule
	Stale   bool
	Warning error
}

type SourceListResult struct {
	Entries []Entry
	Stale   bool
	Warning error
}

type Source interface {
	Name() string
	Lookup(context.Context, RuleID) (LookupResult, error)
	List(context.Context) (SourceListResult, error)
}

type PatternResolver interface {
	Resolve(context.Context, string) (registry.Resolution, error)
}

type Resolution struct {
	Rule     Rule
	Stale    bool
	Warnings []error
}

type ListResult struct {
	Entries  []Entry
	Stale    bool
	Warnings []error
}

type Resolver struct {
	patterns PatternResolver
	sources  []Source
}

func NewResolver(patterns PatternResolver, sources ...Source) (*Resolver, error) {
	if patterns == nil {
		return nil, errors.New("rule resolver requires a pattern resolver")
	}
	if len(sources) == 0 {
		return nil, errors.New("rule resolver requires at least one rule source")
	}
	for index, source := range sources {
		if source == nil {
			return nil, fmt.Errorf("rule resolver source %d is nil", index)
		}
	}
	return &Resolver{patterns: patterns, sources: append([]Source(nil), sources...)}, nil
}

func (r *Resolver) Resolve(ctx context.Context, rawID string) (Resolution, error) {
	id, err := ParseRuleID(rawID)
	if err != nil {
		return Resolution{}, err
	}
	searched := make([]string, 0, len(r.sources))
	warnings := make([]error, 0)
	for _, source := range r.sources {
		if err := ctx.Err(); err != nil {
			return Resolution{}, err
		}
		searched = append(searched, source.Name())
		result, lookupErr := source.Lookup(ctx, id)
		if result.Warning != nil {
			warnings = append(warnings, result.Warning)
		}
		if lookupErr != nil {
			if errors.Is(lookupErr, ErrRuleNotFound) {
				continue
			}
			return Resolution{}, fmt.Errorf("resolve %s from %s: %w", id, source.Name(), lookupErr)
		}
		stale := result.Stale
		seenPatterns := make(map[string]struct{}, len(result.Rule.Definition.Scopes))
		for _, scope := range result.Rule.Definition.Scopes {
			if _, exists := seenPatterns[scope.Pattern]; exists {
				continue
			}
			seenPatterns[scope.Pattern] = struct{}{}
			pattern, patternErr := r.patterns.Resolve(ctx, scope.Pattern)
			if patternErr != nil {
				return Resolution{}, fmt.Errorf("rule %s references unavailable pattern %s: %w", id, scope.Pattern, patternErr)
			}
			stale = stale || pattern.Stale
			warnings = append(warnings, pattern.Warnings...)
		}
		return Resolution{Rule: result.Rule, Stale: stale, Warnings: warnings}, nil
	}
	return Resolution{Warnings: warnings}, fmt.Errorf("%w: %s (searched: %s)", ErrRuleNotFound, id, strings.Join(searched, ", "))
}

func (r *Resolver) List(ctx context.Context) (ListResult, error) {
	entries := make(map[RuleID]Entry)
	warnings := make([]error, 0)
	stale := false
	for _, source := range r.sources {
		if err := ctx.Err(); err != nil {
			return ListResult{}, err
		}
		result, err := source.List(ctx)
		if err != nil {
			return ListResult{}, fmt.Errorf("list rules from %s: %w", source.Name(), err)
		}
		stale = stale || result.Stale
		if result.Warning != nil {
			warnings = append(warnings, result.Warning)
		}
		for _, entry := range result.Entries {
			if _, exists := entries[entry.ID]; !exists {
				entries[entry.ID] = entry
			}
		}
	}
	ordered := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		ordered = append(ordered, entry)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	return ListResult{Entries: ordered, Stale: stale, Warnings: warnings}, nil
}
