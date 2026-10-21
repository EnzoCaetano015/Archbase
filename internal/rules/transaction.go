package rules

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
)

type fileChange struct {
	path              string
	original, updated []byte
	existed, remove   bool
	ids               []string
	legacy            bool
}

func (e *Exporter) read(root, relative string) (fileChange, error) {
	p, err := confinedTarget(root, relative)
	if err != nil {
		return fileChange{}, err
	}
	if err = e.rejectSymlinkPath(root, p); err != nil {
		return fileChange{}, err
	}
	c := fileChange{path: p}
	info, err := e.fs.Lstat(p)
	if errors.Is(err, fs.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, fmt.Errorf("inspect %q: %w", p, err)
	}
	if !info.Mode().IsRegular() {
		return c, fmt.Errorf("target %q must be a regular file", p)
	}
	c.existed = true
	c.original, err = e.fs.ReadFile(p)
	return c, err
}

func (e *Exporter) applyChanges(ctx context.Context, root string, changes map[string]fileChange) ([]string, error) {
	actions := make([]fileChange, 0, len(changes))
	for _, c := range changes {
		if c.remove && !c.existed {
			continue
		}
		if !c.remove && c.existed && bytes.Equal(c.original, c.updated) {
			continue
		}
		actions = append(actions, c)
	}
	sort.Slice(actions, func(i, j int) bool { return actions[i].path < actions[j].path })
	applied := []fileChange{}
	createdDirs := []string{}
	rollback := func(cause error) ([]string, error) {
		failures := []error{cause}
		for i := len(applied) - 1; i >= 0; i-- {
			c := applied[i]
			var err error
			if c.existed {
				err = e.fs.WriteFileAtomic(c.path, c.original, true)
			} else {
				err = e.fs.Remove(c.path)
			}
			if err != nil {
				failures = append(failures, fmt.Errorf("rollback %q: %w", c.path, err))
			}
		}
		for i := len(createdDirs) - 1; i >= 0; i-- {
			if err := e.fs.Remove(createdDirs[i]); err != nil {
				failures = append(failures, fmt.Errorf("rollback directory %q: %w", createdDirs[i], err))
			}
		}
		return nil, errors.Join(failures...)
	}
	for _, c := range actions {
		if err := ctx.Err(); err != nil {
			return rollback(err)
		}
		if !c.remove {
			missing := []string{}
			for dir := filepath.Dir(c.path); dir != root && dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
				exists, err := e.fs.Exists(dir)
				if err != nil {
					return rollback(err)
				}
				if exists {
					break
				}
				missing = append(missing, dir)
			}
			for i := len(missing) - 1; i >= 0; i-- {
				// Include a directory even if its creation reports a late failure.
				createdDirs = append(createdDirs, missing[i])
				if err := e.fs.EnsureDir(missing[i]); err != nil {
					return rollback(err)
				}
			}
		}
		// Include the failing write: filesystem implementations can fail after replacing it.
		applied = append(applied, c)
		var err error
		if c.remove {
			err = e.fs.Remove(c.path)
		} else {
			err = e.fs.WriteFileAtomic(c.path, c.updated, c.existed)
		}
		if err != nil {
			return rollback(fmt.Errorf("update rule file %q: %w", c.path, err))
		}
		if err := ctx.Err(); err != nil {
			return rollback(err)
		}
	}
	paths := make([]string, 0, len(actions))
	for _, c := range actions {
		paths = append(paths, c.path)
	}
	return paths, nil
}
