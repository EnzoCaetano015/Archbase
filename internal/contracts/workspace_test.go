package contracts

import (
	"errors"
	"testing"
	"testing/fstest"
)

func TestWorkspaceContract(t *testing.T) {
	scope := Scope{SchemaVersion: 1, Scope: ScopeSelector{Path: "."}, Pattern: ScopePattern{ID: "next/page@1234", Source: "registry"}, Behavior: ScopeBehavior{NearestScopeWins: true}, Metadata: Metadata{"custom": true}}
	valid := Workspace{SchemaVersion: 1, Scopes: []Scope{scope}, Metadata: Metadata{"project": "example"}}
	data, err := EncodeWorkspace(valid)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadWorkspaceFS(fstest.MapFS{"scopes.yaml": {Data: data}}, "scopes.yaml")
	if err != nil || loaded.Scopes[0].Metadata["custom"] != true {
		t.Fatalf("%#v %v", loaded, err)
	}
	for _, p := range []string{"../escape", "/absolute", "src\\pages", "src/../pages", "C:/outside", ""} {
		invalid := scope
		invalid.Scope.Path = p
		_, err := EncodeWorkspace(Workspace{SchemaVersion: 1, Scopes: []Scope{invalid}})
		var structured *ValidationError
		if !errors.As(err, &structured) || structured.Kind != WorkspaceKind || structured.Field == "" {
			t.Fatalf("%s: %v", p, err)
		}
	}
	for _, scopes := range [][]Scope{{scope, scope}, {func() Scope { v := scope; v.Scope.Path = "z"; return v }(), scope}} {
		if _, err := EncodeWorkspace(Workspace{SchemaVersion: 1, Scopes: scopes}); err == nil {
			t.Fatal("duplicate/unsorted accepted")
		}
	}
	for _, doc := range []string{"schemaVersion: 2\nscopes: []\n", "schemaVersion: 1\nscopes: []\nunknown: true\n", "scopes: []\n"} {
		if _, err := LoadWorkspaceFS(fstest.MapFS{"scopes.yaml": {Data: []byte(doc)}}, "scopes.yaml"); err == nil {
			t.Fatal("invalid accepted")
		}
	}
}
