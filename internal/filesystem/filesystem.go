package filesystem

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrConflict = errors.New("destination already exists")
	ErrSymlink  = errors.New("symbolic links are not allowed")
)

type PathError struct {
	Op   string
	Path string
	Err  error
}

func (e *PathError) Error() string { return fmt.Sprintf("%s %q: %v", e.Op, e.Path, e.Err) }
func (e *PathError) Unwrap() error { return e.Err }

func ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, &PathError{Op: "read", Path: path, Err: err}
	}
	return data, nil
}

func Exists(path string) (bool, error) {
	_, err := os.Lstat(filepath.Clean(path))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, &PathError{Op: "check", Path: path, Err: err}
}

func EnsureDir(path string) error {
	if err := os.MkdirAll(filepath.Clean(path), 0o755); err != nil {
		return &PathError{Op: "create directory", Path: path, Err: err}
	}
	return nil
}

func WriteFileAtomic(path string, data []byte, overwrite bool) error {
	cleanPath := filepath.Clean(path)
	exists, err := Exists(cleanPath)
	if err != nil {
		return err
	}
	if exists {
		info, statErr := os.Lstat(cleanPath)
		if statErr != nil {
			return &PathError{Op: "inspect destination", Path: path, Err: statErr}
		}
		if !info.Mode().IsRegular() {
			return &PathError{Op: "write", Path: path, Err: errors.New("destination is not a regular file")}
		}
	}
	if exists && !overwrite {
		return &PathError{Op: "write", Path: path, Err: ErrConflict}
	}
	parent := filepath.Dir(cleanPath)
	if err := EnsureDir(parent); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(parent, ".archbase-write-*")
	if err != nil {
		return &PathError{Op: "create temporary file", Path: path, Err: err}
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err = temporary.Write(data); err == nil {
		err = temporary.Sync()
	}
	closeErr := temporary.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return &PathError{Op: "write", Path: path, Err: err}
	}
	if err := os.Rename(temporaryPath, cleanPath); err != nil {
		// os.Rename atomically replaces regular files on platforms that support
		// it. Windows rejects an existing destination, so use the safe fallback
		// only after the atomic operation has been attempted.
		if !overwrite || !exists {
			return &PathError{Op: "commit write", Path: path, Err: err}
		}
		if removeErr := os.Remove(cleanPath); removeErr != nil {
			return &PathError{Op: "replace", Path: path, Err: removeErr}
		}
		if renameErr := os.Rename(temporaryPath, cleanPath); renameErr != nil {
			return &PathError{Op: "commit write", Path: path, Err: renameErr}
		}
	}
	return nil
}

type copyEntry struct {
	relative string
	info     fs.FileInfo
}

func CopyTree(source, destination string, overwrite bool) error {
	sourceAbs, err := filepath.Abs(source)
	if err != nil {
		return &PathError{Op: "resolve source", Path: source, Err: err}
	}
	destinationAbs, err := filepath.Abs(destination)
	if err != nil {
		return &PathError{Op: "resolve destination", Path: destination, Err: err}
	}
	var entries []copyEntry
	err = filepath.Walk(sourceAbs, func(current string, info fs.FileInfo, walkErr error) error {
		if walkErr != nil {
			return &PathError{Op: "walk", Path: current, Err: walkErr}
		}
		relative, relErr := filepath.Rel(sourceAbs, current)
		if relErr != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return &PathError{Op: "copy", Path: current, Err: errors.New("path escapes source root")}
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return &PathError{Op: "copy", Path: current, Err: ErrSymlink}
		}
		if relative != "." && !info.IsDir() && !info.Mode().IsRegular() {
			return &PathError{Op: "copy", Path: current, Err: errors.New("unsupported file type")}
		}
		entries = append(entries, copyEntry{relative: relative, info: info})
		return nil
	})
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.relative == "." || entry.info.IsDir() || overwrite {
			continue
		}
		target := filepath.Join(destinationAbs, entry.relative)
		if exists, checkErr := Exists(target); checkErr != nil {
			return checkErr
		} else if exists {
			return &PathError{Op: "copy", Path: target, Err: ErrConflict}
		}
	}
	for _, entry := range entries {
		target := destinationAbs
		if entry.relative != "." {
			target = filepath.Join(destinationAbs, entry.relative)
		}
		if entry.info.IsDir() {
			if err := EnsureDir(target); err != nil {
				return err
			}
			continue
		}
		content, err := os.ReadFile(filepath.Join(sourceAbs, entry.relative))
		if err != nil {
			return &PathError{Op: "read for copy", Path: entry.relative, Err: err}
		}
		if err := WriteFileAtomic(target, content, overwrite); err != nil {
			return err
		}
	}
	return nil
}
