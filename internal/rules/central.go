package rules

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/EnzoCaetano015/Archbase/internal/contracts"
)

func (e *Exporter) loadDocuments(ctx context.Context, root string) (map[string]documentMetadata, map[string]fileChange, error) {
	documents := map[string]documentMetadata{}
	files := map[string]fileChange{}
	dir, err := confinedTarget(root, ".archbase/rules")
	if err != nil {
		return nil, nil, err
	}
	if err = e.rejectSymlinkPath(root, dir); err != nil {
		return nil, nil, err
	}
	info, err := e.fs.Lstat(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return documents, files, nil
	}
	if err != nil {
		return nil, nil, err
	}
	if !info.IsDir() {
		return nil, nil, fmt.Errorf("rules directory %q must be a regular directory", dir)
	}
	err = e.fs.WalkDir(dir, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if p == dir {
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("symbolic links are not allowed at %q", p)
		}
		if d.IsDir() {
			return fs.SkipDir
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		relative, _ := filepath.Rel(root, p)
		relative = filepath.ToSlash(relative)
		c, err := e.read(root, relative)
		if err != nil {
			return err
		}
		_, owned, err := ownedExportRuleID(c.original)
		if err != nil {
			return fmt.Errorf("rule document %q: %w", p, err)
		}
		if !owned {
			return nil
		}
		m, err := readDocument(relative, c.original)
		if err != nil {
			return fmt.Errorf("rule document %q: %w", p, err)
		}
		if _, ok := documents[m.ID]; ok {
			return fmt.Errorf("duplicate installed rule %q", m.ID)
		}
		documents[m.ID], files[m.ID] = m, c
		return nil
	})
	return documents, files, err
}

// scanLegacy never follows directories belonging to dependencies or nested projects.
func (e *Exporter) scanLegacy(ctx context.Context, root, id string, all bool) (map[string]fileChange, error) {
	changes := map[string]fileChange{}
	err := e.fs.WalkDir(root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if p == root {
			return nil
		}
		relative, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".archbase", "node_modules", ".next", ".venv", "vendor", "dist", "build", "coverage", "bin", "obj":
				return fs.SkipDir
			}
			if _, err := e.fs.Lstat(filepath.Join(p, ".git")); err == nil {
				return fs.SkipDir
			} else if !errors.Is(err, fs.ErrNotExist) {
				return err
			}
			return nil
		}
		principal := false
		for _, entry := range entryPaths {
			principal = principal || relative == entry
		}
		instruction := isAgentsPath(relative) || filepath.Base(p) == "CLAUDE.md"
		if !instruction && !principal && !isOwnedExportPath(relative) {
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("symbolic link at managed candidate %q", p)
		}
		c, err := e.read(root, relative)
		if err != nil {
			return err
		}
		if instruction || principal {
			if _, _, _, err = indexRange(c.original); err != nil {
				return fmt.Errorf("instruction file %q: %w", p, err)
			}
			updated, ids, changed, err := removeManagedBlocks(c.original, id, all)
			if err != nil {
				return fmt.Errorf("instruction file %q: %w", p, err)
			}
			if changed {
				c.updated, c.ids = updated, ids
				c.remove = !principal && len(bytes.TrimSpace(updated)) == 0
				changes[p] = c
			}
			return nil
		}
		found, owned, err := ownedExportRuleID(c.original)
		if err != nil {
			return fmt.Errorf("legacy export %q: %w", p, err)
		}
		if owned && (all || found == id) {
			if relative != expectedExportPath(found, relative) {
				return fmt.Errorf("legacy export filename and ownership ID disagree at %q", p)
			}
			c.remove = true
			c.ids = []string{found}
			changes[p] = c
		} else if !owned && isLegacyManagedExport(c.original, relative) && (all || relative == expectedExportPath(id, relative)) {
			c.remove, c.legacy = true, true
			if !all {
				c.ids = []string{id}
			}
			changes[p] = c
		}
		return nil
	})
	return changes, err
}

func (e *Exporter) synchronizeIndices(root string, selected Format, documents map[string]documentMetadata, changes map[string]fileChange) error {
	for _, format := range formats {
		c, err := e.read(root, entryPaths[format])
		if err != nil {
			return err
		}
		_, _, active, err := indexRange(c.original)
		if err != nil {
			return fmt.Errorf("index %q: %w", c.path, err)
		}
		if !active && format != selected {
			continue
		}
		if previous, ok := changes[c.path]; ok {
			c = previous
		}
		if c.existed && format == FormatCursor {
			if err = validateCursor(c.original); err != nil {
				return fmt.Errorf("Cursor index %q: %w", c.path, err)
			}
		}
		existing := c.original
		if c.updated != nil {
			existing = c.updated
		}
		block := renderIndex(format, documents)
		if !c.existed {
			c.updated = newEntry(format, block)
		} else {
			c.updated, err = replaceIndex(existing, block)
			if err != nil {
				return err
			}
		}
		c.remove = false
		changes[c.path] = c
	}
	return nil
}

func (e *Exporter) prepareInstallation(ctx context.Context, root string, rule Rule, format Format, options ExportOptions) (map[string]fileChange, error) {
	artifact, err := renderDocument(rule)
	if err != nil {
		return nil, err
	}
	m, err := readDocument(artifact.RelativePath, artifact.Content)
	if err != nil {
		return nil, err
	}
	documents, _, err := e.loadDocuments(ctx, root)
	if err != nil {
		return nil, err
	}
	target, err := e.read(root, artifact.RelativePath)
	if err != nil {
		return nil, err
	}
	if target.existed {
		previous, err := readDocument(artifact.RelativePath, target.original)
		if err != nil {
			return nil, fmt.Errorf("refusing to replace external or mismatched document %q: %w", target.path, err)
		}
		if previous.ID != m.ID {
			return nil, fmt.Errorf("rule filename collision at %q", target.path)
		}
		if !bytes.Equal(target.original, artifact.Content) && !options.Overwrite {
			return nil, fmt.Errorf("rule document %q already exists with different content; use --overwrite", target.path)
		}
	}
	changes, err := e.scanLegacy(ctx, root, m.ID, false)
	if err != nil {
		return nil, err
	}
	target.updated, target.ids = artifact.Content, []string{m.ID}
	changes[target.path] = target
	documents[m.ID] = m
	index, err := e.read(root, ".archbase/scopes.yaml")
	if err != nil {
		return nil, err
	}
	if index.existed {
		if _, err = contracts.LoadWorkspaceBytes(index.path, index.original); err != nil {
			return nil, err
		}
	} else {
		index.updated, err = contracts.EncodeWorkspace(contracts.Workspace{SchemaVersion: 1, Scopes: []contracts.Scope{}})
		if err != nil {
			return nil, err
		}
		changes[index.path] = index
	}
	if err = e.synchronizeIndices(root, format, documents, changes); err != nil {
		return nil, err
	}
	return changes, nil
}
