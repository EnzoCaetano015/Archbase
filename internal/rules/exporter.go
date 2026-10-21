package rules

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	archfs "github.com/EnzoCaetano015/Archbase/internal/filesystem"
)

type ExportFileSystem interface {
	Exists(string) (bool, error)
	EnsureDir(string) error
	ReadFile(string) ([]byte, error)
	WriteFileAtomic(string, []byte, bool) error
	Lstat(string) (fs.FileInfo, error)
	Remove(string) error
	WalkDir(string, fs.WalkDirFunc) error
	CreateExclusive(string) error
}

type OSExportFileSystem struct{}

func (OSExportFileSystem) CreateExclusive(p string) error { return archfs.CreateExclusive(p) }

func (OSExportFileSystem) Exists(value string) (bool, error) { return archfs.Exists(value) }
func (OSExportFileSystem) EnsureDir(value string) error      { return archfs.EnsureDir(value) }
func (OSExportFileSystem) ReadFile(value string) ([]byte, error) {
	return archfs.ReadFile(value)
}
func (OSExportFileSystem) WriteFileAtomic(value string, data []byte, overwrite bool) error {
	return archfs.WriteFileAtomic(value, data, overwrite)
}
func (OSExportFileSystem) Lstat(value string) (fs.FileInfo, error) { return os.Lstat(value) }
func (OSExportFileSystem) WalkDir(root string, walk fs.WalkDirFunc) error {
	return filepath.WalkDir(root, walk)
}
func (OSExportFileSystem) Remove(value string) error {
	if err := os.Remove(value); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return &archfs.PathError{Op: "remove", Path: value, Err: err}
	}
	return nil
}

type ExportOptions struct {
	Destination string
	Overwrite   bool
	Merge       bool
}

type ExportResult struct {
	Format Format
	Paths  []string
}

type Exporter struct{ fs ExportFileSystem }

func NewExporter(fsys ExportFileSystem) (*Exporter, error) {
	if fsys == nil {
		return nil, errors.New("rules exporter filesystem is required")
	}
	return &Exporter{fs: fsys}, nil
}

func (e *Exporter) Export(rule Rule, format Format, options ExportOptions) (ExportResult, error) {
	return e.ExportContext(context.Background(), rule, format, options)
}

func (e *Exporter) ExportContext(ctx context.Context, rule Rule, format Format, options ExportOptions) (ExportResult, error) {
	if _, err := ParseFormat(string(format)); err != nil {
		return ExportResult{}, err
	}
	if options.Destination == "" {
		options.Destination = "."
	}
	root, err := filepath.Abs(options.Destination)
	if err != nil {
		return ExportResult{}, err
	}
	if err = e.validateRoot(root); err != nil {
		return ExportResult{}, err
	}
	if err = ctx.Err(); err != nil {
		return ExportResult{}, err
	}
	if err = e.fs.EnsureDir(root); err != nil {
		return ExportResult{}, err
	}
	unlock, err := archfs.AcquireWorkspaceLock(ctx, root, e.fs)
	if err != nil {
		return ExportResult{}, err
	}
	defer unlock()
	changes, err := e.prepareInstallation(ctx, root, rule, format, options)
	if err != nil {
		return ExportResult{}, err
	}
	paths, err := e.applyChanges(ctx, root, changes)
	return ExportResult{Format: format, Paths: paths}, err
}

func (e *Exporter) validateRoot(root string) error {
	exists, err := e.fs.Exists(root)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	info, err := e.fs.Lstat(root)
	if err != nil {
		return &archfs.PathError{Op: "inspect export destination", Path: root, Err: err}
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return &archfs.PathError{Op: "export", Path: root, Err: archfs.ErrSymlink}
	}
	if !info.IsDir() {
		return &archfs.PathError{Op: "export", Path: root, Err: errors.New("destination is not a directory")}
	}
	return nil
}

func (e *Exporter) rejectSymlinkPath(root, target string) error {
	relative, _ := filepath.Rel(root, target)
	current := root
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		current = filepath.Join(current, component)
		exists, err := e.fs.Exists(current)
		if err != nil {
			return err
		}
		if !exists {
			return nil
		}
		info, err := e.fs.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return &archfs.PathError{Op: "export", Path: current, Err: archfs.ErrSymlink}
		}
	}
	return nil
}

func confinedTarget(root, relative string) (string, error) {
	hasDrivePrefix := len(relative) >= 2 && relative[1] == ':' && ((relative[0] >= 'a' && relative[0] <= 'z') || (relative[0] >= 'A' && relative[0] <= 'Z'))
	if relative == "" || path.IsAbs(relative) || filepath.IsAbs(relative) || hasDrivePrefix || strings.Contains(relative, "\\") || path.Clean(relative) != relative {
		return "", &archfs.PathError{Op: "export", Path: relative, Err: errors.New("invalid relative export path")}
	}
	clean := filepath.Clean(filepath.FromSlash(relative))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", &archfs.PathError{Op: "export", Path: relative, Err: errors.New("path escapes export destination")}
	}
	target := filepath.Join(root, clean)
	resolved, err := filepath.Rel(root, target)
	if err != nil || resolved == ".." || strings.HasPrefix(resolved, ".."+string(filepath.Separator)) {
		return "", &archfs.PathError{Op: "export", Path: relative, Err: errors.New("path escapes export destination")}
	}
	return target, nil
}
