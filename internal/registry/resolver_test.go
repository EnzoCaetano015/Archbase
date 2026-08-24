package registry

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

type stubSource struct {
	name   string
	lookup func(context.Context, PatternID) (LookupResult, error)
}

func (s stubSource) Name() string { return s.name }
func (s stubSource) Lookup(ctx context.Context, id PatternID) (LookupResult, error) {
	return s.lookup(ctx, id)
}
func (s stubSource) List(context.Context) (ListResult, error) { return ListResult{}, nil }

func TestResolverUsesFirstMatchingSource(t *testing.T) {
	id, _ := ParsePatternID("test/item@1")
	missing := stubSource{name: "first", lookup: func(context.Context, PatternID) (LookupResult, error) {
		return LookupResult{}, fmt.Errorf("%w: first", ErrPatternNotFound)
	}}
	match := stubSource{name: "second", lookup: func(context.Context, PatternID) (LookupResult, error) {
		return LookupResult{Pattern: Pattern{Entry: Entry{ID: id}, Source: "second"}}, nil
	}}
	resolver, err := NewResolver(missing, match)
	if err != nil {
		t.Fatal(err)
	}
	result, err := resolver.Resolve(context.Background(), id.String())
	if err != nil || result.Pattern.Source != "second" {
		t.Fatalf("unexpected resolution: %#v, %v", result, err)
	}
}

func TestResolverStopsOnCorruptedSource(t *testing.T) {
	corrupted := stubSource{name: "corrupted", lookup: func(context.Context, PatternID) (LookupResult, error) {
		return LookupResult{}, errors.New("invalid registry index")
	}}
	fallbackCalled := false
	fallback := stubSource{name: "fallback", lookup: func(context.Context, PatternID) (LookupResult, error) {
		fallbackCalled = true
		return LookupResult{}, nil
	}}
	resolver, _ := NewResolver(corrupted, fallback)
	_, err := resolver.Resolve(context.Background(), "test/item@1")
	if err == nil || !strings.Contains(err.Error(), "invalid registry index") || fallbackCalled {
		t.Fatalf("expected hard failure without fallback, got %v", err)
	}
}

func TestResolverReturnsWarningsAndNotFoundDetails(t *testing.T) {
	id, _ := ParsePatternID("test/item@1")
	stale := stubSource{name: "stale", lookup: func(context.Context, PatternID) (LookupResult, error) {
		return LookupResult{Pattern: Pattern{Entry: Entry{ID: id}}, Stale: true, Warning: errors.New("offline")}, nil
	}}
	resolver, _ := NewResolver(stale)
	result, err := resolver.Resolve(context.Background(), id.String())
	if err != nil || !result.Stale || len(result.Warnings) != 1 {
		t.Fatalf("unexpected stale result: %#v, %v", result, err)
	}

	missing := stubSource{name: "embedded", lookup: func(context.Context, PatternID) (LookupResult, error) {
		return LookupResult{}, ErrPatternNotFound
	}}
	resolver, _ = NewResolver(missing)
	_, err = resolver.Resolve(context.Background(), "test/missing@1")
	if !errors.Is(err, ErrPatternNotFound) || !strings.Contains(err.Error(), "embedded") {
		t.Fatalf("expected detailed not-found error, got %v", err)
	}
}

func TestResolverPreservesWarningFromMissingEarlierSource(t *testing.T) {
	id, _ := ParsePatternID("test/item@1")
	earlier := stubSource{name: "remote", lookup: func(context.Context, PatternID) (LookupResult, error) {
		return LookupResult{Stale: true, Warning: errors.New("remote unavailable")}, ErrPatternNotFound
	}}
	later := stubSource{name: "embedded", lookup: func(context.Context, PatternID) (LookupResult, error) {
		return LookupResult{Pattern: Pattern{Entry: Entry{ID: id}, Source: "embedded"}}, nil
	}}
	resolver, _ := NewResolver(earlier, later)
	result, err := resolver.Resolve(context.Background(), id.String())
	if err != nil || result.Stale || len(result.Warnings) != 1 || result.Pattern.Source != "embedded" {
		t.Fatalf("unexpected fallback result: %#v, %v", result, err)
	}
}

func TestResolverValidatesConstructionAndID(t *testing.T) {
	if _, err := NewResolver(); err == nil {
		t.Fatal("expected empty resolver error")
	}
	if _, err := NewResolver(nil); err == nil {
		t.Fatal("expected nil source error")
	}
	resolver, _ := NewResolver(stubSource{name: "source", lookup: func(context.Context, PatternID) (LookupResult, error) {
		return LookupResult{}, nil
	}})
	if _, err := resolver.Resolve(context.Background(), "Invalid"); err == nil {
		t.Fatal("expected invalid ID error")
	}
}
