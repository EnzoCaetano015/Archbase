package skillbundle

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/EnzoCaetano015/Archbase/internal/patterns"
)

// loadFixture reads typed Markdown documents into their logical project paths.
// For example, app/page.tsx.md represents app/page.tsx; a Markdown project file
// such as AGENTS.md is stored as AGENTS.md.md and remains inert in the repository.
func loadFixture(root, fixture string) (map[string][]byte, error) {
	dir, err := confinedFile(root, fixture)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("fixture is not a directory: %s", fixture)
	}
	files := map[string][]byte{}
	seen := map[string]bool{}
	err = filepath.WalkDir(dir, func(file string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("fixture symlink: %s", file)
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, file)
		if err != nil {
			return err
		}
		document := filepath.ToSlash(rel)
		if err = safePath(document); err != nil {
			return err
		}
		if !strings.HasSuffix(document, ".md") {
			return fmt.Errorf("%s: fixture sources must be typed Markdown documents", document)
		}
		logical := strings.TrimSuffix(document, ".md")
		if err = safePath(logical); err != nil {
			return err
		}
		if seen[strings.ToLower(logical)] {
			return fmt.Errorf("fixture path collision: %s", logical)
		}
		seen[strings.ToLower(logical)] = true
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || info.Size() > MaxFileBytes {
			return fmt.Errorf("%s: invalid fixture document", document)
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		code, _, err := patterns.DecodeDocument(document, data)
		if err != nil {
			return err
		}
		files[logical] = code
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("empty fixture %s", fixture)
	}
	return files, nil
}

func fixtureDigest(root, fixture string) (string, error) {
	files, err := loadFixture(root, fixture)
	if err != nil {
		return "", err
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var tree strings.Builder
	for _, name := range names {
		fmt.Fprintf(&tree, "%s  %s\n", digest(files[name]), name)
	}
	return digest([]byte(tree.String())), nil
}

// MaterializeFixture extracts one scenario's project into a new directory. It
// never executes project content or writes into the versioned fixture directory.
func MaterializeFixture(suiteFile, id, destination string) (err error) {
	suite, err := LoadSuite(suiteFile)
	if err != nil {
		return err
	}
	var selected *Scenario
	for i := range suite.Scenarios {
		if suite.Scenarios[i].ID == id {
			selected = &suite.Scenarios[i]
			break
		}
	}
	if selected == nil {
		return fmt.Errorf("unknown scenario %s", id)
	}
	files, err := loadFixture(filepath.Dir(suiteFile), selected.Fixture)
	if err != nil {
		return err
	}
	destination, err = filepath.Abs(destination)
	if err != nil {
		return err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(destination))
	if err != nil {
		return err
	}
	destination = filepath.Join(parent, filepath.Base(destination))
	suiteRoot, err := filepath.Abs(filepath.Dir(suiteFile))
	if err != nil {
		return err
	}
	for _, scenario := range suite.Scenarios {
		source, err := filepath.EvalSymlinks(filepath.Join(suiteRoot, filepath.FromSlash(scenario.Fixture)))
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, destination)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel) {
			return fmt.Errorf("materialization destination must remain outside versioned fixtures")
		}
	}
	if err = os.Mkdir(destination, 0755); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = os.RemoveAll(destination)
		}
	}()
	for name, data := range files {
		if err = writeNew(filepath.Join(destination, filepath.FromSlash(name)), data); err != nil {
			return err
		}
	}
	return nil
}
