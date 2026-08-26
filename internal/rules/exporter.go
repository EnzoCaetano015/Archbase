package rules

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
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
}

type OSExportFileSystem struct{}

func (OSExportFileSystem) Exists(value string) (bool, error) { return archfs.Exists(value) }
func (OSExportFileSystem) EnsureDir(value string) error      { return archfs.EnsureDir(value) }
func (OSExportFileSystem) ReadFile(value string) ([]byte, error) {
	return archfs.ReadFile(value)
}
func (OSExportFileSystem) WriteFileAtomic(value string, data []byte, overwrite bool) error {
	return archfs.WriteFileAtomic(value, data, overwrite)
}
func (OSExportFileSystem) Lstat(value string) (fs.FileInfo, error) { return os.Lstat(value) }
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
	if options.Destination == "" {
		options.Destination = "."
	}
	if format == FormatAgents && options.Overwrite {
		return ExportResult{}, errors.New("--overwrite is not supported for agents; use --merge")
	}
	if format != FormatAgents && options.Merge {
		return ExportResult{}, fmt.Errorf("--merge is only supported for agents, not %s", format)
	}
	artifacts, err := Render(rule, format)
	if err != nil {
		return ExportResult{}, err
	}
	paths, err := e.writeArtifacts(options.Destination, artifacts, format, options)
	if err != nil {
		return ExportResult{}, err
	}
	return ExportResult{Format: format, Paths: paths}, nil
}

type preparedArtifact struct {
	path     string
	content  []byte
	existed  bool
	original []byte
}

func (e *Exporter) writeArtifacts(destination string, artifacts []Artifact, format Format, options ExportOptions) ([]string, error) {
	root, err := filepath.Abs(destination)
	if err != nil {
		return nil, &archfs.PathError{Op: "resolve export destination", Path: destination, Err: err}
	}
	if err := e.validateRoot(root); err != nil {
		return nil, err
	}
	prepared := make([]preparedArtifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		target, err := confinedTarget(root, artifact.RelativePath)
		if err != nil {
			return nil, err
		}
		if err := e.rejectSymlinkPath(root, target); err != nil {
			return nil, err
		}
		item := preparedArtifact{path: target, content: artifact.Content}
		item.existed, err = e.fs.Exists(target)
		if err != nil {
			return nil, err
		}
		if item.existed {
			info, statErr := e.fs.Lstat(target)
			if statErr != nil {
				return nil, &archfs.PathError{Op: "inspect export target", Path: target, Err: statErr}
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return nil, &archfs.PathError{Op: "export", Path: target, Err: archfs.ErrSymlink}
			}
			if !info.Mode().IsRegular() {
				return nil, &archfs.PathError{Op: "export", Path: target, Err: errors.New("destination is not a regular file")}
			}
			item.original, err = e.fs.ReadFile(target)
			if err != nil {
				return nil, err
			}
			if format == FormatAgents {
				if !options.Merge {
					return nil, &archfs.PathError{Op: "export", Path: target, Err: errors.New("AGENTS.md exists; use --merge")}
				}
				item.content, err = mergeManagedBlock(item.original, artifact)
				if err != nil {
					return nil, &archfs.PathError{Op: "merge", Path: target, Err: err}
				}
			} else if !options.Overwrite {
				return nil, &archfs.PathError{Op: "export", Path: target, Err: archfs.ErrConflict}
			}
		}
		prepared = append(prepared, item)
	}
	if err := e.fs.EnsureDir(root); err != nil {
		return nil, err
	}
	written := make([]preparedArtifact, 0, len(prepared))
	for _, item := range prepared {
		writeErr := e.fs.EnsureDir(filepath.Dir(item.path))
		if writeErr == nil {
			writeErr = e.fs.WriteFileAtomic(item.path, item.content, item.existed)
		}
		if writeErr != nil {
			rollbackSet := written
			// An overwrite implementation may have replaced or removed the
			// current destination before reporting its commit failure.
			if item.existed {
				rollbackSet = append(append([]preparedArtifact(nil), written...), item)
			}
			rollbackErr := e.rollback(rollbackSet)
			if rollbackErr != nil {
				return nil, fmt.Errorf("export %q: %w (rollback failed: %v)", item.path, writeErr, rollbackErr)
			}
			return nil, fmt.Errorf("export %q: %w", item.path, writeErr)
		}
		written = append(written, item)
	}
	paths := make([]string, 0, len(prepared))
	for _, item := range prepared {
		paths = append(paths, item.path)
	}
	sort.Strings(paths)
	return paths, nil
}

func (e *Exporter) rollback(written []preparedArtifact) error {
	var failures []error
	for index := len(written) - 1; index >= 0; index-- {
		item := written[index]
		var err error
		if item.existed {
			err = e.fs.WriteFileAtomic(item.path, item.original, true)
		} else {
			err = e.fs.Remove(item.path)
		}
		if err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func (e *Exporter) validateRoot(root string) error {
	if err := e.rejectExistingAncestorSymlinks(root); err != nil {
		return err
	}
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

func (e *Exporter) rejectExistingAncestorSymlinks(value string) error {
	current := filepath.Clean(value)
	for {
		exists, err := e.fs.Exists(current)
		if err != nil {
			return err
		}
		if exists {
			info, err := e.fs.Lstat(current)
			if err != nil {
				return err
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return &archfs.PathError{Op: "export", Path: current, Err: archfs.ErrSymlink}
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
		current = parent
	}
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

func mergeManagedBlock(existing []byte, artifact Artifact) ([]byte, error) {
	if artifact.StartMarker == "" || artifact.EndMarker == "" {
		return nil, errors.New("managed block markers are required")
	}
	text := string(existing)
	startCount := strings.Count(text, artifact.StartMarker)
	endCount := strings.Count(text, artifact.EndMarker)
	if startCount == 0 && endCount == 0 {
		separator := ""
		if len(existing) > 0 {
			separator = "\n"
			if !strings.HasSuffix(text, "\n") {
				separator = "\n\n"
			}
		}
		return []byte(text + separator + string(artifact.Content)), nil
	}
	if startCount != 1 || endCount != 1 {
		return nil, errors.New("managed block markers are incomplete or duplicated")
	}
	start := strings.Index(text, artifact.StartMarker)
	end := strings.Index(text, artifact.EndMarker)
	if end < start {
		return nil, errors.New("managed block markers are out of order")
	}
	end += len(artifact.EndMarker)
	if end < len(text) && text[end] == '\r' {
		end++
	}
	if end < len(text) && text[end] == '\n' {
		end++
	}
	return []byte(text[:start] + string(artifact.Content) + text[end:]), nil
}
