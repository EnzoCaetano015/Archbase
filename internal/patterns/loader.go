package patterns

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/EnzoCaetano015/Archbase/internal/contracts"
)

var (
	ErrRequiredFileMissing  = errors.New("required pattern file is missing")
	ErrDuplicateSource      = errors.New("duplicate pattern source")
	ErrDuplicateDestination = errors.New("duplicate pattern destination")
	ErrSymlink              = errors.New("symbolic links are not allowed in patterns")
	ErrNotRegular           = errors.New("pattern source is not a regular file")
)

type Error struct {
	Manifest string
	Field    string
	Path     string
	Err      error
}

func (e *Error) Error() string {
	field := ""
	if e.Field != "" {
		field = " field " + e.Field
	}
	return fmt.Sprintf("invalid pattern %q%s at %q: %v", e.Manifest, field, e.Path, e.Err)
}

func (e *Error) Unwrap() error { return e.Err }

type File struct {
	Spec    contracts.PatternFile
	Present bool
	Content []byte
}

type Bundle struct {
	Manifest contracts.Manifest
	Files    []File
}

// Load validates and reads a complete pattern rooted at root in fsys.
func Load(fsys fs.FS, root string) (Bundle, error) {
	cleanRoot, err := cleanRoot(root)
	if err != nil {
		return Bundle{}, err
	}
	patternFS := fsys
	manifestPath := "manifest.yaml"
	if cleanRoot != "." {
		if err := rejectSymlinkComponents(fsys, path.Join(cleanRoot, manifestPath)); err != nil {
			return Bundle{}, &Error{Manifest: path.Join(cleanRoot, manifestPath), Path: cleanRoot, Err: err}
		}
		patternFS, err = fs.Sub(fsys, cleanRoot)
		if err != nil {
			return Bundle{}, &Error{Manifest: path.Join(cleanRoot, manifestPath), Path: cleanRoot, Err: err}
		}
	}
	manifestDisplayPath := path.Join(cleanRoot, manifestPath)
	manifest, err := contracts.LoadManifestFS(patternFS, manifestPath)
	if err != nil {
		return Bundle{}, err
	}
	result := Bundle{Manifest: manifest, Files: make([]File, 0, len(manifest.Structure.Files))}
	sources := make(map[string]int, len(manifest.Structure.Files))
	destinations := make(map[string]int, len(manifest.Structure.Files))
	for index, spec := range manifest.Structure.Files {
		source := normalizePath(spec.Source)
		destination := normalizePath(spec.Destination)
		if previous, exists := sources[source]; exists {
			return Bundle{}, &Error{Manifest: manifestDisplayPath, Field: fmt.Sprintf("/structure/files/%d/source", index), Path: spec.Source, Err: fmt.Errorf("%w; first declared at index %d", ErrDuplicateSource, previous)}
		}
		if previous, exists := destinations[destination]; exists {
			return Bundle{}, &Error{Manifest: manifestDisplayPath, Field: fmt.Sprintf("/structure/files/%d/destination", index), Path: spec.Destination, Err: fmt.Errorf("%w; first declared at index %d", ErrDuplicateDestination, previous)}
		}
		sources[source] = index
		destinations[destination] = index
		file := File{Spec: spec}
		if err := rejectSymlinkComponents(patternFS, source); err != nil {
			if errors.Is(err, fs.ErrNotExist) && !spec.Required {
				result.Files = append(result.Files, file)
				continue
			}
			if errors.Is(err, fs.ErrNotExist) {
				err = ErrRequiredFileMissing
			}
			return Bundle{}, &Error{Manifest: manifestDisplayPath, Field: fmt.Sprintf("/structure/files/%d/source", index), Path: spec.Source, Err: err}
		}
		info, statErr := fs.Stat(patternFS, source)
		if statErr != nil {
			if errors.Is(statErr, fs.ErrNotExist) && !spec.Required {
				result.Files = append(result.Files, file)
				continue
			}
			if errors.Is(statErr, fs.ErrNotExist) {
				statErr = ErrRequiredFileMissing
			}
			return Bundle{}, &Error{Manifest: manifestDisplayPath, Field: fmt.Sprintf("/structure/files/%d/source", index), Path: spec.Source, Err: statErr}
		}
		if !info.Mode().IsRegular() {
			return Bundle{}, &Error{Manifest: manifestDisplayPath, Field: fmt.Sprintf("/structure/files/%d/source", index), Path: spec.Source, Err: ErrNotRegular}
		}
		content, readErr := fs.ReadFile(patternFS, source)
		if readErr != nil {
			return Bundle{}, &Error{Manifest: manifestDisplayPath, Field: fmt.Sprintf("/structure/files/%d/source", index), Path: spec.Source, Err: readErr}
		}
		file.Present = true
		file.Content = content
		result.Files = append(result.Files, file)
	}
	return result, nil
}

func cleanRoot(root string) (string, error) {
	root = normalizePath(root)
	cleaned := path.Clean(root)
	if cleaned == "" || cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.HasPrefix(cleaned, "/") {
		return "", &Error{Manifest: path.Join(root, "manifest.yaml"), Path: root, Err: errors.New("pattern root escapes registry")}
	}
	return cleaned, nil
}

func normalizePath(value string) string { return strings.ReplaceAll(value, "\\", "/") }

func rejectSymlinkComponents(fsys fs.FS, filePath string) error {
	parts := strings.Split(filePath, "/")
	current := "."
	for _, part := range parts {
		entries, err := fs.ReadDir(fsys, current)
		if err != nil {
			return err
		}
		found := false
		for _, entry := range entries {
			if entry.Name() != part {
				continue
			}
			found = true
			if entry.Type()&fs.ModeSymlink != 0 {
				return ErrSymlink
			}
			break
		}
		if !found {
			return fs.ErrNotExist
		}
		current = path.Join(current, part)
	}
	return nil
}
