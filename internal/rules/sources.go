package rules

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/EnzoCaetano015/Archbase/internal/registry"
	registrydata "github.com/EnzoCaetano015/Archbase/registry"
)

func NewEmbeddedSource() (Source, error) {
	rulesFS, err := fs.Sub(registrydata.FS, "rules")
	if err != nil {
		return nil, fmt.Errorf("open embedded rule registry: %w", err)
	}
	return newCatalogSource("official-embedded", rulesFS)
}

func NewDirectorySource(root string) (Source, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve rule registry directory %q: %w", root, err)
	}
	info, err := os.Lstat(absolute)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("open rule registry directory %q: not a regular directory", absolute)
	}
	rulesRoot := filepath.Join(absolute, "rules")
	info, err = os.Lstat(rulesRoot)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("open rule registry catalog %q: not a regular directory", rulesRoot)
	}
	return newCatalogSource("directory:"+absolute, os.DirFS(rulesRoot))
}

type gitSource struct {
	provider registry.CheckoutProvider
}

func NewGitSource(config registry.GitSourceConfig) (Source, error) {
	provider, err := registry.NewGitSource(config)
	if err != nil {
		return nil, err
	}
	return newGitSource(provider), nil
}

func newGitSource(provider registry.CheckoutProvider) Source {
	return &gitSource{provider: provider}
}

func (s *gitSource) Name() string { return s.provider.Name() }

func (s *gitSource) Lookup(ctx context.Context, id RuleID) (LookupResult, error) {
	catalog, snapshot, missing, err := s.catalog(ctx)
	if err != nil {
		return LookupResult{}, err
	}
	if missing {
		return LookupResult{Stale: snapshot.Stale, Warning: snapshot.Warning}, fmt.Errorf("rule registry %s: %w: %s", s.Name(), ErrRuleNotFound, id)
	}
	result, err := catalog.Lookup(ctx, id)
	result.Stale = snapshot.Stale
	result.Warning = snapshot.Warning
	if err == nil {
		result.Rule.Source = s.Name()
		result.Rule.Entry.Source = s.Name()
	}
	return result, err
}

func (s *gitSource) List(ctx context.Context) (SourceListResult, error) {
	catalog, snapshot, missing, err := s.catalog(ctx)
	if err != nil {
		return SourceListResult{}, err
	}
	if missing {
		return SourceListResult{Stale: snapshot.Stale, Warning: snapshot.Warning}, nil
	}
	result, err := catalog.List(ctx)
	result.Stale = snapshot.Stale
	result.Warning = snapshot.Warning
	return result, err
}

func (s *gitSource) catalog(ctx context.Context) (*catalogSource, registry.CheckoutSnapshot, bool, error) {
	snapshot, err := s.provider.Snapshot(ctx)
	if err != nil {
		return nil, registry.CheckoutSnapshot{}, false, err
	}
	info, statErr := fs.Stat(snapshot.FS, "rules")
	if statErr != nil {
		if errors.Is(statErr, fs.ErrNotExist) {
			return nil, snapshot, true, nil
		}
		return nil, snapshot, false, fmt.Errorf("inspect Git rule catalog: %w", statErr)
	}
	if !info.IsDir() {
		return nil, snapshot, false, errors.New("Git rule catalog is not a directory")
	}
	rulesFS, err := fs.Sub(snapshot.FS, "rules")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, snapshot, true, nil
		}
		return nil, snapshot, false, fmt.Errorf("open Git rule catalog: %w", err)
	}
	catalog, err := newCatalogSource(s.Name(), rulesFS)
	if err != nil {
		if snapshot.Stale {
			return nil, snapshot, false, fmt.Errorf("validate stale rule registry cache: %w", err)
		}
		return nil, snapshot, false, err
	}
	return catalog, snapshot, false, nil
}
