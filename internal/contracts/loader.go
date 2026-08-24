package contracts

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/EnzoCaetano015/Archbase/schemas"
	"github.com/goccy/go-yaml"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

type Kind string

const (
	ManifestKind Kind = "manifest"
	ScopeKind    Kind = "scope"
	RuleKind     Kind = "rule"
)

var schemaFiles = map[Kind]string{
	ManifestKind: "manifest.schema.json",
	ScopeKind:    "scope.schema.json",
	RuleKind:     "rule.schema.json",
}

type ValidationError struct {
	Kind  Kind
	Path  string
	Field string
	Cause error
}

func (e *ValidationError) Error() string {
	field := ""
	if e.Field != "" {
		field = " field " + e.Field
	}
	return fmt.Sprintf("invalid %s %q%s: %v", e.Kind, e.Path, field, e.Cause)
}

func (e *ValidationError) Unwrap() error { return e.Cause }

func LoadManifest(path string) (Manifest, error) {
	var result Manifest
	err := loadFile(ManifestKind, path, &result)
	if err == nil {
		err = validateManifestPaths(path, result)
	}
	return result, err
}

func LoadManifestFS(fsys fs.FS, path string) (Manifest, error) {
	var result Manifest
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return result, &ValidationError{Kind: ManifestKind, Path: path, Cause: err}
	}
	err = decodeAndValidate(ManifestKind, path, data, &result)
	if err == nil {
		err = validateManifestPaths(path, result)
	}
	return result, err
}

func LoadScope(path string) (Scope, error) {
	var result Scope
	err := loadFile(ScopeKind, path, &result)
	return result, err
}

func LoadRule(path string) (Rule, error) {
	var result Rule
	err := loadFile(RuleKind, path, &result)
	return result, err
}

func loadFile(kind Kind, path string, destination any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return &ValidationError{Kind: kind, Path: path, Cause: err}
	}
	return decodeAndValidate(kind, path, data, destination)
}

func decodeAndValidate(kind Kind, sourcePath string, data []byte, destination any) error {
	var yamlValue any
	if err := yaml.Unmarshal(data, &yamlValue); err != nil {
		return &ValidationError{Kind: kind, Path: sourcePath, Cause: fmt.Errorf("parse YAML: %w", err)}
	}
	jsonData, err := json.Marshal(yamlValue)
	if err != nil {
		return &ValidationError{Kind: kind, Path: sourcePath, Cause: fmt.Errorf("convert YAML to JSON: %w", err)}
	}
	if err := validateJSON(kind, jsonData); err != nil {
		return &ValidationError{Kind: kind, Path: sourcePath, Field: validationField(err), Cause: err}
	}
	if err := yaml.Unmarshal(data, destination); err != nil {
		return &ValidationError{Kind: kind, Path: sourcePath, Cause: fmt.Errorf("decode YAML: %w", err)}
	}
	return nil
}

func validateJSON(kind Kind, data []byte) error {
	schemaName, ok := schemaFiles[kind]
	if !ok {
		return fmt.Errorf("unsupported document kind %q", kind)
	}
	schemaData, err := schemas.FS.ReadFile(schemaName)
	if err != nil {
		return fmt.Errorf("read embedded schema: %w", err)
	}
	schemaValue, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaData))
	if err != nil {
		return fmt.Errorf("parse embedded schema: %w", err)
	}
	compiler := jsonschema.NewCompiler()
	resourceURL := "https://archbase.dev/embedded/" + schemaName
	if err := compiler.AddResource(resourceURL, schemaValue); err != nil {
		return fmt.Errorf("register embedded schema: %w", err)
	}
	compiled, err := compiler.Compile(resourceURL)
	if err != nil {
		return fmt.Errorf("compile embedded schema: %w", err)
	}
	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("parse JSON instance: %w", err)
	}
	return compiled.Validate(instance)
}

func validationField(err error) string {
	var validationErr *jsonschema.ValidationError
	if errors.As(err, &validationErr) {
		if len(validationErr.InstanceLocation) == 0 {
			return "/"
		}
		parts := make([]string, len(validationErr.InstanceLocation))
		for index, part := range validationErr.InstanceLocation {
			part = strings.ReplaceAll(part, "~", "~0")
			parts[index] = strings.ReplaceAll(part, "/", "~1")
		}
		return "/" + strings.Join(parts, "/")
	}
	return ""
}

func validateManifestPaths(sourcePath string, manifest Manifest) error {
	for index, file := range manifest.Structure.Files {
		for field, value := range map[string]string{"source": file.Source, "destination": file.Destination} {
			if !safeRelativePath(value) {
				return &ValidationError{
					Kind:  ManifestKind,
					Path:  sourcePath,
					Field: fmt.Sprintf("/structure/files/%d/%s", index, field),
					Cause: fmt.Errorf("path must be relative and remain inside the pattern root: %q", value),
				}
			}
		}
	}
	return nil
}

func safeRelativePath(value string) bool {
	if value == "" || filepath.IsAbs(value) || filepath.VolumeName(value) != "" {
		return false
	}
	normalized := strings.ReplaceAll(value, "\\", "/")
	cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(normalized)))
	return cleaned != ".." && !strings.HasPrefix(cleaned, "../") && cleaned != "."
}
