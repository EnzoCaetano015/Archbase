package contracts

import (
	"fmt"
	"io/fs"
	"path"
	"strings"
)

func LoadWorkspaceFS(fsys fs.FS, file string) (Workspace, error) {
	var result Workspace
	data, err := fs.ReadFile(fsys, file)
	if err != nil {
		return result, &ValidationError{Kind: WorkspaceKind, Path: file, Cause: err}
	}
	if err = decodeAndValidate(WorkspaceKind, file, data, &result); err == nil {
		err = validateWorkspace(file, result)
	}
	return result, err
}

func EncodeWorkspace(value Workspace) ([]byte, error) {
	if err := validateWorkspace("scopes.yaml", value); err != nil {
		return nil, err
	}
	return encodeAndValidate(WorkspaceKind, "scopes.yaml", value)
}

func validateWorkspace(file string, value Workspace) error {
	previous := ""
	for i, scope := range value.Scopes {
		p := scope.Scope.Path
		if p == "" || path.Clean(p) != p || strings.ContainsAny(p, "\\:") || strings.HasPrefix(p, "/") || p == ".." || strings.HasPrefix(p, "../") || p == ".archbase" || strings.HasPrefix(p, ".archbase/") || (i > 0 && p <= previous) {
			return &ValidationError{Kind: WorkspaceKind, Path: file, Field: fmt.Sprintf("/scopes/%d/scope/path", i), Cause: fmt.Errorf("scope paths must be confined, normalized, unique and sorted: %q", p)}
		}
		previous = p
		if scope.Pattern.Source == "local" {
			r := scope.Pattern.Root
			if r == "" || r == "." || r == ".." || path.Clean(r) != r || strings.ContainsAny(r, "\\:") || strings.HasPrefix(r, "/") || strings.HasPrefix(r, "../") {
				return &ValidationError{Kind: WorkspaceKind, Path: file, Field: fmt.Sprintf("/scopes/%d/pattern/root", i), Cause: fmt.Errorf("local pattern root must remain inside .archbase: %q", r)}
			}
		}
	}
	return nil
}
