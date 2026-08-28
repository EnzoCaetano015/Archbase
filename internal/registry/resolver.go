package registry

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

type Resolution struct {
	Pattern  Pattern
	Stale    bool
	Warnings []error
}

type ResolutionList struct {
	Entries  []Entry
	Stale    bool
	Warnings []error
}

type Resolver struct {
	sources []Source
}

func NewResolver(sources ...Source) (*Resolver, error) {
	if len(sources) == 0 {
		return nil, fmt.Errorf("resolver requires at least one registry source")
	}
	for index, source := range sources {
		if source == nil {
			return nil, fmt.Errorf("resolver source %d is nil", index)
		}
	}
	return &Resolver{sources: append([]Source(nil), sources...)}, nil
}

func (r *Resolver) Resolve(ctx context.Context, rawID string) (Resolution, error) {
	id, err := ParsePatternID(rawID)
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
		result, err := source.Lookup(ctx, id)
		if result.Warning != nil {
			warnings = append(warnings, result.Warning)
		}
		if err != nil {
			if errors.Is(err, ErrPatternNotFound) {
				continue
			}
			return Resolution{}, fmt.Errorf("resolve %s from %s: %w", id, source.Name(), err)
		}
		return Resolution{Pattern: result.Pattern, Stale: result.Stale, Warnings: warnings}, nil
	}
	return Resolution{Warnings: warnings}, fmt.Errorf("%w: %s (searched: %s)", ErrPatternNotFound, id, strings.Join(searched, ", "))
}

// List combines ordered sources, keeping the first occurrence of each ID.
func (r *Resolver) List(ctx context.Context) (ResolutionList, error) {
	entries := make(map[PatternID]Entry)
	warnings := make([]error, 0)
	stale := false
	for _, source := range r.sources {
		if err := ctx.Err(); err != nil {
			return ResolutionList{}, err
		}
		result, err := source.List(ctx)
		if err != nil {
			return ResolutionList{}, fmt.Errorf("list patterns from %s: %w", source.Name(), err)
		}
		stale = stale || result.Stale
		if result.Warning != nil {
			warnings = append(warnings, result.Warning)
		}
		for _, entry := range result.Entries {
			if _, exists := entries[entry.ID]; exists {
				continue
			}
			entry.Source = source.Name()
			entries[entry.ID] = entry
		}
	}
	ordered := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		ordered = append(ordered, entry)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	return ResolutionList{Entries: ordered, Stale: stale, Warnings: warnings}, nil
}
