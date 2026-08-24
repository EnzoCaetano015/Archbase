package registry

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type Resolution struct {
	Pattern  Pattern
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
