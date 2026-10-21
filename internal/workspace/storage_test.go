package workspace

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/EnzoCaetano015/Archbase/internal/patterns"
)

func TestReadableStorageNames(t *testing.T) {
	for _, test := range []struct{ scope, id, want string }{
		{".", "next/page@1234", "patterns/_root/next--page--1234"},
		{"features/storefront/application", "next-feature/application-service@3527", "patterns/features--storefront--application/next-feature--application-service--3527"},
		{"src/pages", "local/pages-standard@1", "patterns/src--pages/local--pages-standard--1"},
	} {
		if got := patternStorageRoot(test.scope, test.id); got != test.want {
			t.Fatalf("%s: %s != %s", test.scope, got, test.want)
		}
	}
}

func TestStorageLabelsDoNotAliasOnCaseInsensitiveFilesystems(t *testing.T) {
	seen := map[string]string{}
	for _, scope := range []string{".", "_root", "src/pages", "src--pages", "Src/pages", "src/Pages", "src-/pages", "src/-pages", "src./pages", "src~2e/pages", "con", "con.txt", "aux", "nul", "com1", "lpt9", "~63on", "feature with spaces", "features/café", "features/cafe"} {
		label := scopeStorageLabel(scope)
		if strings.ContainsAny(label, "/\\:") || strings.HasSuffix(label, ".") || strings.Contains(label, " ") {
			t.Fatal("unsafe label", label)
		}
		if old, ok := seen[strings.ToLower(label)]; ok {
			t.Fatalf("%q aliases %q as %q", scope, old, label)
		}
		seen[strings.ToLower(label)] = scope
	}
	if patternStorageRoot(".", "a/b-c@1") == patternStorageRoot(".", "a-b/c@1") {
		t.Fatal("PatternIDs alias")
	}
	first := scopeStorageLabel(strings.Repeat("feature/", 50) + "a")
	second := scopeStorageLabel(strings.Repeat("feature/", 50) + "b")
	if len(first) > 120 || first == second || !strings.HasPrefix(first, "feature--") {
		t.Fatal("long labels", first, second)
	}
}

func TestHashedBundlesRemainReadableAndProtected(t *testing.T) {
	s, root := newWorkspace(t, OSFileSystem{})
	ctx := context.Background()
	installed, err := s.Add(ctx, "next/page@1234", root)
	if err != nil {
		t.Fatal(err)
	}
	index := loadIndex(t, root)
	oldKey := fmt.Sprintf("patterns/%x/%x", sha256.Sum256([]byte(".")), sha256.Sum256([]byte("next/page@1234")))
	oldRoot := filepath.Join(root, ".archbase", filepath.FromSlash(oldKey))
	if err = os.MkdirAll(filepath.Dir(oldRoot), 0755); err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(installed.PatternDirectory, oldRoot); err != nil {
		t.Fatal(err)
	}
	index.Scopes[0].Pattern.Root = oldKey
	saveIndex(t, root, index)
	custom := []byte("export const customized = true;\n")
	if err = os.WriteFile(filepath.Join(oldRoot, "Example", "Example.tsx.md"), patterns.EncodeDocument("Example/Example.tsx", custom), 0644); err != nil {
		t.Fatal(err)
	}
	resolved, err := s.Resolve(ctx, root)
	if err != nil || string(resolved.Pattern.Bundle.Files[0].Content) != string(custom) {
		t.Fatalf("old customization lost: %v", err)
	}
	if _, err = s.Add(ctx, "next/page@1234", root); !errors.Is(err, ErrPatternExists) {
		t.Fatalf("active collision: %v", err)
	}
	if _, err = s.Add(ctx, "next/hook@9214", root); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Add(ctx, "next/page@1234", root); !errors.Is(err, ErrPatternExists) {
		t.Fatalf("inactive collision: %v", err)
	}
	if _, err = s.Add(ctx, "next/page@1234", filepath.Join(root, "src", "pages")); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(root, ".archbase", "patterns", "src--pages", "next--page--1234", "manifest.yaml")); err != nil {
		t.Fatal(err)
	}
}
