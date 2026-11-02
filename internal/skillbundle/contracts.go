// Package skillbundle implements development-only skill delivery contracts.
// It is not part of the arc CLI or a public Go API.
package skillbundle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/EnzoCaetano015/Archbase/schemas"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const CatalogRepository = "https://github.com/EnzoCaetano015/Archbase"

type Manifest struct {
	SchemaVersion int      `json:"schemaVersion"`
	Name          string   `json:"name"`
	Version       string   `json:"version"`
	License       string   `json:"license"`
	Files         []string `json:"files"`
	Rules         []string `json:"rules"`
}

type LockedFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type CatalogEntry struct {
	ID       string   `json:"id"`
	Version  string   `json:"version"`
	Path     string   `json:"path"`
	Patterns []string `json:"patterns"`
}

type CatalogLock struct {
	SchemaVersion int            `json:"schemaVersion"`
	Repository    string         `json:"repository"`
	Tag           string         `json:"tag"`
	Commit        string         `json:"commit"`
	Files         []LockedFile   `json:"files"`
	Rules         []CatalogEntry `json:"rules"`
}

var semver = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?$`)
var skillName = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
var commitID = regexp.MustCompile(`^[a-f0-9]{40}$`)

func decodeContract(data []byte, schemaName string, target any) error {
	compiler := jsonschema.NewCompiler()
	schemaData, err := schemas.FS.ReadFile(schemaName)
	if err != nil {
		return err
	}
	resource, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaData))
	if err != nil {
		return err
	}
	url := "https://archbase.dev/schemas/v1/" + schemaName
	if err = compiler.AddResource(url, resource); err != nil {
		return err
	}
	schema, err := compiler.Compile(url)
	if err != nil {
		return err
	}
	value, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return err
	}
	if err = schema.Validate(value); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(target); err != nil {
		return err
	}
	if err = decoder.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("trailing JSON data")
	}
	return nil
}

func readContract(file, schema string, target any) error {
	data, err := os.ReadFile(file)
	if err == nil {
		err = decodeContract(data, schema, target)
	}
	if err != nil {
		return fmt.Errorf("%s: %w", file, err)
	}
	return nil
}

// safePath applies the same portable spelling to inputs and ZIP entries.
func safePath(value string) error {
	if value == "" || path.Clean(value) != value || path.IsAbs(value) || strings.ContainsAny(value, "\\:\x00?#%<>\"|*") {
		return fmt.Errorf("unsafe relative path %q", value)
	}
	for _, character := range value {
		if character < 32 || character == 127 {
			return fmt.Errorf("unsafe control character in path %q", value)
		}
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." || strings.HasSuffix(part, ".") || strings.HasSuffix(part, " ") {
			return fmt.Errorf("unsafe path component %q", part)
		}
		base := strings.ToUpper(strings.SplitN(part, ".", 2)[0])
		if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || regexp.MustCompile(`^(COM|LPT)[0-9]$`).MatchString(base) {
			return fmt.Errorf("non-portable path %q", value)
		}
	}
	return nil
}

func confinedFile(root, name string) (string, error) {
	if err := safePath(name); err != nil {
		return "", err
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	// Inspect the root's ancestors as well; a symlinked root is not a boundary.
	for current := abs; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("symlink: %s", current)
		}
		if current == filepath.Dir(current) {
			break
		}
	}
	current := abs
	for _, part := range strings.Split(name, "/") {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("symlink: %s", current)
		}
	}
	return current, nil
}

func writeNew(file string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	closeErr := f.Close()
	if err != nil {
		_ = os.Remove(file)
		return err
	}
	if closeErr != nil {
		_ = os.Remove(file)
	}
	return closeErr
}

func WriteJSON(file string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return writeNew(file, append(data, '\n'))
}
