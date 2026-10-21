package filesystem

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

type LockFileSystem interface {
	CreateExclusive(string) error
	Lstat(string) (fs.FileInfo, error)
	Remove(string) error
}

func CreateExclusive(p string) error {
	f, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		_ = os.Remove(p)
	}
	return err
}

// AcquireWorkspaceLock serializes pattern and rule transactions in the same project.
// A crashed writer's lock is never stolen; normal exits always release it.
func AcquireWorkspaceLock(ctx context.Context, root string, fsys LockFileSystem) (func(), error) {
	p := filepath.Join(root, ".archbase.lock")
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	retries := 0
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		err := fsys.CreateExclusive(p)
		if err == nil {
			return func() { _ = fsys.Remove(p) }, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			if runtime.GOOS != "windows" || !errors.Is(err, fs.ErrPermission) || retries >= 10 {
				return nil, fmt.Errorf("lock workspace %q: %w", p, err)
			}
			retries++
		} else {
			retries = 0
			info, e := fsys.Lstat(p)
			if e != nil && !errors.Is(e, fs.ErrNotExist) {
				return nil, fmt.Errorf("inspect lock %q: %w", p, e)
			}
			if e == nil && !info.Mode().IsRegular() {
				return nil, fmt.Errorf("workspace lock %q must be a regular file, not a symlink or directory", p)
			}
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}
