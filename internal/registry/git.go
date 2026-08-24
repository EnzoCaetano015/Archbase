package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	archfs "github.com/EnzoCaetano015/Archbase/internal/filesystem"
	git "github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
)

const (
	cacheLockPollInterval = 25 * time.Millisecond
	cacheLockMaxAge       = 30 * time.Minute
)

var errCacheLockHeld = errors.New("registry cache lock is held")

type gitCacheState struct {
	LastFetch time.Time `json:"lastFetch"`
	Reference string    `json:"reference"`
}

type GitSource struct {
	config GitSourceConfig
	name   string
	now    func() time.Time
}

func NewGitSource(config GitSourceConfig) (*GitSource, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &GitSource{
		config: config,
		name:   "git:" + config.cacheKey()[:12],
		now:    time.Now,
	}, nil
}

func (s *GitSource) Name() string { return s.name }

func (s *GitSource) Lookup(ctx context.Context, id PatternID) (LookupResult, error) {
	catalog, stale, warning, err := s.catalog(ctx)
	if err != nil {
		return LookupResult{}, err
	}
	result, err := catalog.Lookup(ctx, id)
	if err != nil {
		return LookupResult{Stale: stale, Warning: warning}, err
	}
	result.Pattern.Source = s.name
	result.Stale = stale
	result.Warning = warning
	return result, nil
}

func (s *GitSource) List(ctx context.Context) (ListResult, error) {
	catalog, stale, warning, err := s.catalog(ctx)
	if err != nil {
		return ListResult{}, err
	}
	result, err := catalog.List(ctx)
	if err != nil {
		return ListResult{}, err
	}
	result.Stale = stale
	result.Warning = warning
	return result, nil
}

func (s *GitSource) catalog(ctx context.Context) (*catalogSource, bool, error, error) {
	refreshErr := s.refresh(ctx)
	if refreshErr != nil && (errors.Is(refreshErr, context.Canceled) || errors.Is(refreshErr, context.DeadlineExceeded)) {
		return nil, false, nil, refreshErr
	}
	cached, cacheErr := NewDirectorySource(s.config.registryPath())
	if cacheErr != nil {
		if refreshErr != nil {
			return nil, false, nil, fmt.Errorf("refresh registry and open cache: %w", errors.Join(refreshErr, cacheErr))
		}
		return nil, false, nil, cacheErr
	}
	catalog, ok := cached.(*catalogSource)
	if !ok {
		return nil, false, nil, fmt.Errorf("internal error: directory registry has unexpected type %T", cached)
	}
	if refreshErr != nil {
		warning := fmt.Errorf("registry update failed; using validated stale cache: %w", refreshErr)
		return catalog, true, warning, nil
	}
	return catalog, false, nil, nil
}

func (s *GitSource) refresh(ctx context.Context) error {
	release, err := s.acquireLock(ctx)
	if err != nil {
		return err
	}
	defer release()

	state, stateErr := s.readState()
	checkoutExists := directoryExists(s.config.checkoutPath())
	if stateErr == nil && checkoutExists && s.now().Sub(state.LastFetch) < s.config.cacheTTL() {
		return nil
	}
	if !checkoutExists {
		return s.clone(ctx)
	}
	if stateErr != nil || state.Reference == "" {
		reference, err := resolveRemoteReference(ctx, s.config.URL, s.config.Ref)
		if err != nil {
			return err
		}
		state.Reference = reference.String()
	}
	return s.update(ctx, state.Reference)
}

func (s *GitSource) clone(ctx context.Context) error {
	reference, err := resolveRemoteReference(ctx, s.config.URL, s.config.Ref)
	if err != nil {
		return fmt.Errorf("resolve registry ref %q: %w", s.config.Ref, err)
	}
	base := filepath.Dir(s.config.checkoutPath())
	if err := os.MkdirAll(base, 0o700); err != nil {
		return fmt.Errorf("create registry cache directory: %w", err)
	}
	temporaryRoot, err := os.MkdirTemp(base, "clone-*")
	if err != nil {
		return fmt.Errorf("create temporary registry checkout: %w", err)
	}
	defer os.RemoveAll(temporaryRoot)
	temporaryCheckout := filepath.Join(temporaryRoot, "checkout")
	_, err = git.PlainCloneContext(ctx, temporaryCheckout, false, &git.CloneOptions{
		URL:           s.config.URL,
		ReferenceName: reference,
		SingleBranch:  true,
		Depth:         1,
		Tags:          git.NoTags,
	})
	if err != nil {
		return fmt.Errorf("clone registry: %w", err)
	}
	if err := os.Rename(temporaryCheckout, s.config.checkoutPath()); err != nil {
		return fmt.Errorf("promote registry checkout: %w", err)
	}
	return s.writeState(gitCacheState{LastFetch: s.now().UTC(), Reference: reference.String()})
}

func (s *GitSource) update(ctx context.Context, reference string) error {
	repository, err := git.PlainOpen(s.config.checkoutPath())
	if err != nil {
		return fmt.Errorf("open cached Git registry: %w", err)
	}
	targetReference := plumbing.ReferenceName("refs/archbase/cache-target")
	refspec := gitconfig.RefSpec("+" + reference + ":" + targetReference.String())
	err = repository.FetchContext(ctx, &git.FetchOptions{
		RemoteName: "origin",
		RefSpecs:   []gitconfig.RefSpec{refspec},
		Force:      true,
		Tags:       git.NoTags,
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("fetch registry: %w", err)
	}
	target, err := repository.Reference(targetReference, true)
	if err != nil {
		return fmt.Errorf("resolve fetched registry ref: %w", err)
	}
	commitHash, err := peelCommit(repository, target.Hash())
	if err != nil {
		return fmt.Errorf("resolve registry commit: %w", err)
	}
	worktree, err := repository.Worktree()
	if err != nil {
		return fmt.Errorf("open registry worktree: %w", err)
	}
	if err := worktree.Reset(&git.ResetOptions{Mode: git.HardReset, Commit: commitHash}); err != nil {
		return fmt.Errorf("reset registry checkout: %w", err)
	}
	return s.writeState(gitCacheState{LastFetch: s.now().UTC(), Reference: reference})
}

func resolveRemoteReference(ctx context.Context, remoteURL, requested string) (plumbing.ReferenceName, error) {
	remote := git.NewRemote(memory.NewStorage(), &gitconfig.RemoteConfig{Name: "origin", URLs: []string{remoteURL}})
	references, err := remote.ListContext(ctx, &git.ListOptions{})
	if err != nil {
		return "", err
	}
	candidates := []plumbing.ReferenceName{plumbing.ReferenceName(requested)}
	if !strings.HasPrefix(requested, "refs/") {
		candidates = []plumbing.ReferenceName{
			plumbing.NewBranchReferenceName(requested),
			plumbing.NewTagReferenceName(requested),
		}
	}
	for _, candidate := range candidates {
		for _, reference := range references {
			if reference.Name() == candidate {
				return candidate, nil
			}
		}
	}
	return "", fmt.Errorf("branch or tag %q was not found", requested)
}

func peelCommit(repository *git.Repository, hash plumbing.Hash) (plumbing.Hash, error) {
	if _, err := repository.CommitObject(hash); err == nil {
		return hash, nil
	}
	tag, err := repository.TagObject(hash)
	if err != nil {
		return plumbing.ZeroHash, err
	}
	return peelTag(repository, tag)
}

func peelTag(repository *git.Repository, tag *object.Tag) (plumbing.Hash, error) {
	if _, err := repository.CommitObject(tag.Target); err == nil {
		return tag.Target, nil
	}
	nested, err := repository.TagObject(tag.Target)
	if err != nil {
		return plumbing.ZeroHash, err
	}
	return peelTag(repository, nested)
}

func (s *GitSource) statePath() string {
	return filepath.Join(filepath.Dir(s.config.checkoutPath()), "state.json")
}

func (s *GitSource) readState() (gitCacheState, error) {
	var state gitCacheState
	data, err := os.ReadFile(s.statePath())
	if err != nil {
		return state, err
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return state, err
	}
	return state, nil
}

func (s *GitSource) writeState(state gitCacheState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	if err := archfs.WriteFileAtomic(s.statePath(), data, true); err != nil {
		return fmt.Errorf("write registry cache state: %w", err)
	}
	return nil
}

func (s *GitSource) lockPath() string {
	return filepath.Join(s.config.CacheRoot, s.config.cacheKey()+".lock")
}

func (s *GitSource) acquireLock(ctx context.Context) (func(), error) {
	if err := os.MkdirAll(s.config.CacheRoot, 0o700); err != nil {
		return nil, fmt.Errorf("create registry cache root: %w", err)
	}
	lockPath := s.lockPath()
	for {
		lock, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			_, _ = fmt.Fprintf(lock, "%d\n", time.Now().UnixNano())
			_ = lock.Close()
			return func() { _ = os.Remove(lockPath) }, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("acquire registry cache lock: %w", err)
		}
		if info, statErr := os.Stat(lockPath); statErr == nil && time.Since(info.ModTime()) > cacheLockMaxAge {
			if removeErr := os.Remove(lockPath); removeErr == nil || errors.Is(removeErr, os.ErrNotExist) {
				continue
			}
		}
		timer := time.NewTimer(cacheLockPollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, fmt.Errorf("%w: %w", errCacheLockHeld, ctx.Err())
		case <-timer.C:
		}
	}
}

func directoryExists(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink == 0 && info.IsDir()
}
