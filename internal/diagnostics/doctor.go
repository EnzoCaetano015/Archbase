package diagnostics

import (
	"context"
	"errors"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/EnzoCaetano015/Archbase/internal/contracts"
	"github.com/EnzoCaetano015/Archbase/internal/patterns"
	"github.com/EnzoCaetano015/Archbase/internal/registry"
	"github.com/EnzoCaetano015/Archbase/internal/rules"
)

// Options lets tests inject denied reads, cancellation and budgets. CLI users
// cannot replace readers or increase the fixed safety budgets.
type Options struct {
	Limits Limits
	Open   func(context.Context, string, Limits) (Reader, error)
}

var checkIDs = []string{"cli-version", "project-root", "workspace-index", "scope-paths", "local-bundles", "scope-origin", "rule-documents", "managed-indices", "legacy-indices", "snapshot"}

func Run(ctx context.Context, root, version string, options Options) Result {
	if version == "" {
		version = "dev"
	}
	result := Result{SchemaVersion: 1, Command: "doctor", ToolVersion: version, ProjectRoot: ".", Checks: []Check{}}
	add := func(id, status, p, message, reason string) {
		var relative *string
		if p != "" && p != "." && safeName(p) && !excluded(p) {
			v := p
			relative = &v
		}
		c := Check{ID: "doctor." + id, Status: status, Path: relative, Message: message, Reason: reason}
		if status == "error" || status == "warning" {
			c.Recommendation = "Review the reported configuration in a separately authorized edit; no changes were made."
		}
		result.Checks = append(result.Checks, c)
	}
	add("cli-version", "passed", "", "Invoked CLI version reported without a network lookup.", "")
	if options.Limits.Files <= 0 {
		options.Limits = DefaultLimits
	}
	if options.Open == nil {
		options.Open = OpenRoot
	}
	if root == "" {
		root = "."
	}
	r, err := options.Open(ctx, root, options.Limits)
	if err != nil {
		result.State = "error"
		add("project-root", "error", "", "Project root could not be opened safely.", reason(err))
		for _, id := range checkIDs[2:] {
			add(id, "not-run", "", "Check blocked by the unavailable project root.", reason(err))
		}
		result.finish()
		return result
	}
	defer r.Close()
	add("project-root", "passed", "", "Project root is a regular confined directory.", "")
	data, workspaceErr := r.ReadFile(".archbase/scopes.yaml")
	var workspace contracts.Workspace
	if workspaceErr == nil {
		workspace, workspaceErr = contracts.LoadWorkspaceBytes(".archbase/scopes.yaml", data)
	}
	if errors.Is(workspaceErr, fs.ErrNotExist) {
		add("workspace-index", "warning", ".archbase/scopes.yaml", "Workspace is not configured.", "missing-workspace")
		for _, id := range []string{"scope-paths", "local-bundles", "scope-origin"} {
			add(id, "not-applicable", "", "No configured scopes to inspect.", "unconfigured-workspace")
		}
	} else if workspaceErr != nil {
		add("workspace-index", failureStatus(workspaceErr), ".archbase/scopes.yaml", "Workspace index could not be validated.", reason(workspaceErr))
		for _, id := range []string{"scope-paths", "local-bundles", "scope-origin"} {
			add(id, "not-run", ".archbase/scopes.yaml", "Scope inspection blocked by the invalid workspace.", reason(workspaceErr))
		}
	} else {
		add("workspace-index", "passed", ".archbase/scopes.yaml", "Workspace index validates under the existing YAML contract.", "")
		var scopeErr error
		for _, scope := range workspace.Scopes {
			if err := ctx.Err(); err != nil {
				scopeErr = err
				break
			}
			// Missing future scope directories are valid; existing components must be safe.
			info, e := r.Lstat(scope.Scope.Path)
			if e == nil && !info.IsDir() {
				e = ErrBoundary
			}
			if e != nil && !errors.Is(e, fs.ErrNotExist) {
				scopeErr = e
				break
			}
		}
		if scopeErr != nil {
			add("scope-paths", failureStatus(scopeErr), ".archbase/scopes.yaml", "Scope path validation failed.", reason(scopeErr))
		} else {
			add("scope-paths", "passed", ".archbase/scopes.yaml", "Scope paths are normalized, unique, sorted and confined.", "")
		}
		inspectBundles(ctx, r, workspace, scopeErr, add)
	}
	documents, documentsErr := inspectRules(r, add)
	inspectIndices(r, documents, documentsErr, add)
	legacy, legacyErr := scanLegacy(r, ".")
	if legacyErr != nil {
		add("legacy-indices", failureStatus(legacyErr), "", "Legacy-index scan could not finish.", reason(legacyErr))
	} else if legacy {
		add("legacy-indices", "warning", "", "Unsupported per-directory scope index exists; it was not loaded or migrated.", "legacy-configuration")
	} else {
		add("legacy-indices", "passed", "", "No unsupported per-directory scope index found in the inspected tree.", "")
	}
	if err := r.Verify(); err != nil {
		add("snapshot", "not-run", "", "Filesystem snapshot could not be verified as stable.", reason(err))
	} else {
		add("snapshot", "passed", "", "Inspected file and directory identities and metadata stayed stable during the diagnostic.", "")
	}
	result.finish()
	return result
}

type addCheck func(string, string, string, string, string)

func reason(err error) string {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return "cancelled"
	case errors.Is(err, ErrBudget):
		return "read-budget-exhausted"
	case errors.Is(err, ErrExcluded):
		return "excluded-sensitive-path"
	case errors.Is(err, ErrChanged):
		return "filesystem-changed"
	case errors.Is(err, ErrBoundary):
		return "unsafe-path-or-link"
	case errors.Is(err, patterns.ErrSymlink), errors.Is(err, patterns.ErrNotRegular):
		return "unsafe-path-or-link"
	case errors.Is(err, fs.ErrPermission):
		return "permission-denied"
	case errors.Is(err, fs.ErrNotExist), errors.Is(err, patterns.ErrRequiredFileMissing):
		return "required-path-missing"
	case errors.Is(err, registry.ErrPatternNotFound):
		return "unknown-origin-pattern"
	default:
		return "invalid-configuration"
	}
}
func failureStatus(err error) string {
	switch reason(err) {
	case "cancelled", "read-budget-exhausted", "excluded-sensitive-path", "filesystem-changed", "permission-denied":
		return "not-run"
	default:
		return "error"
	}
}

func inspectBundles(ctx context.Context, r Reader, workspace contracts.Workspace, scopeErr error, add addCheck) {
	if scopeErr != nil {
		for _, id := range []string{"local-bundles", "scope-origin"} {
			add(id, "not-run", ".archbase/scopes.yaml", "Bundle inspection blocked by unsafe scope paths.", reason(scopeErr))
		}
		return
	}
	var bundleErr, originErr error
	bundlePath := ".archbase/scopes.yaml"
	count, origins := 0, 0
	verifiedOrigins := 0
	var embedded registry.Source
	for _, scope := range workspace.Scopes {
		if err := ctx.Err(); err != nil {
			bundleErr = err
			break
		}
		if scope.Origin != nil {
			origins++
			id, parseErr := registry.ParsePatternID(scope.Origin.ID)
			if parseErr != nil || strings.TrimSpace(scope.Origin.Registry) == "" {
				originErr = errors.New("invalid origin")
			} else if scope.Origin.Registry == "official-embedded" {
				if embedded == nil {
					var initializationErr error
					embedded, initializationErr = registry.NewEmbeddedSource()
					if initializationErr != nil {
						originErr = initializationErr
					}
				}
				if embedded != nil {
					found, err := embedded.Lookup(ctx, id)
					if err != nil {
						originErr = err
					} else if found.Pattern.Bundle.Manifest.Version == scope.Origin.Version {
						verifiedOrigins++
					}
				}
			}
			// A remote origin cannot be verified offline. Never instantiate a Git source.
		}
		if scope.Pattern.Source != "local" {
			continue
		}
		count++
		local := path.Join(".archbase", scope.Pattern.Root)
		bundlePath = path.Join(local, "manifest.yaml")
		bundle, err := patterns.Load(boundedFS{r}, local)
		if err == nil && bundle.Manifest.ID != scope.Pattern.ID {
			err = errors.New("bundle identity mismatch")
		}
		if err != nil {
			var patternError *patterns.Error
			if errors.As(err, &patternError) && safeName(patternError.Path) && !excluded(patternError.Path) {
				if strings.HasPrefix(patternError.Path, local+"/") {
					bundlePath = patternError.Path
				} else if patternError.Path != local && !strings.HasPrefix(patternError.Path, ".archbase/") {
					bundlePath = path.Join(local, patternError.Path)
				}
			}
			bundleErr = err
			break
		}
	}
	if bundleErr != nil {
		add("local-bundles", failureStatus(bundleErr), bundlePath, "Local bundle validation failed; no ancestor fallback was used.", reason(bundleErr))
	} else if count == 0 {
		add("local-bundles", "not-applicable", "", "No local bundles are configured.", "no-local-bundles")
	} else {
		add("local-bundles", "passed", "", "Local bundle identities, manifests and declared files validate.", "")
	}
	if bundleErr != nil {
		add("scope-origin", "not-run", "", "Origin inspection blocked by incomplete bundle validation.", reason(bundleErr))
	} else if originErr != nil {
		add("scope-origin", "error", ".archbase/scopes.yaml", "Declared origin metadata is invalid.", reason(originErr))
	} else if origins == 0 {
		add("scope-origin", "not-applicable", "", "No origins declared; local customization does not require one.", "no-declared-origin")
	} else if origins == verifiedOrigins {
		add("scope-origin", "passed", ".archbase/scopes.yaml", "Declared origins exist in the embedded catalog at their recorded versions; local customizations were not compared with source bytes.", "")
	} else {
		add("scope-origin", "warning", ".archbase/scopes.yaml", "Origin metadata is structurally valid; remote identity was not verified offline.", "origin-not-verified-offline")
	}
}

func inspectRules(r Reader, add addCheck) (map[string][]byte, error) {
	documents := map[string][]byte{}
	entries, err := r.ReadDir(".archbase/rules")
	if errors.Is(err, fs.ErrNotExist) {
		add("rule-documents", "not-applicable", "", "No central rule documents are installed.", "no-installed-rules")
		return documents, nil
	}
	foreign := false
	failurePath := ".archbase/rules"
	if err == nil {
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			name := path.Join(".archbase/rules", entry.Name())
			data, e := r.ReadFile(name)
			if e != nil {
				failurePath = name
				err = e
				break
			}
			_, owned, e := rules.InspectDocument(name, data)
			if e != nil {
				failurePath = name
				err = e
				break
			}
			if !owned {
				foreign = true
				continue
			}
			documents[name] = data
		}
	}
	if err != nil {
		add("rule-documents", failureStatus(err), failurePath, "Central rule document validation failed.", reason(err))
	} else if foreign {
		add("rule-documents", "warning", ".archbase/rules", "Owned documents validate; foreign documents were not treated as owned.", "foreign-rule-documents")
	} else if len(documents) == 0 {
		add("rule-documents", "not-applicable", "", "No owned central rule documents are installed.", "no-installed-rules")
	} else {
		add("rule-documents", "passed", ".archbase/rules", "Installed owned rule documents validate.", "")
	}
	return documents, err
}

func inspectIndices(r Reader, documents map[string][]byte, blocked error, add addCheck) {
	if blocked != nil {
		add("managed-indices", "not-run", "", "Index comparison blocked by incomplete owned rule validation.", reason(blocked))
		return
	}
	entries := []struct {
		path   string
		format rules.Format
	}{{"AGENTS.md", rules.FormatAgents}, {"CLAUDE.md", rules.FormatClaude}, {".cursor/rules/archbase.mdc", rules.FormatCursor}, {".github/copilot-instructions.md", rules.FormatCopilot}}
	active := false
	for _, entry := range entries {
		data, err := r.ReadFile(entry.path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			add("managed-indices", failureStatus(err), entry.path, "Native instruction index could not be inspected.", reason(err))
			return
		}
		on, err := rules.InspectIndex(entry.format, data, documents)
		if err != nil {
			add("managed-indices", "error", entry.path, "Managed block is malformed, stale or incompatible with owned rules.", "invalid-managed-index")
			return
		}
		active = active || on
	}
	if active {
		add("managed-indices", "passed", "", "Active managed references match installed owned documents; external bytes were preserved.", "")
	} else {
		add("managed-indices", "not-applicable", "", "No managed native instruction indexes are active.", "no-active-indices")
	}
}

func scanLegacy(r Reader, dir string) (bool, error) {
	entries, err := r.ReadDir(dir)
	if err != nil {
		return false, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		name := path.Join(dir, entry.Name())
		if entry.Name() == ".archbase" {
			_, e := r.Lstat(path.Join(name, "scope.yaml"))
			if e == nil {
				return true, nil
			}
			if !errors.Is(e, fs.ErrNotExist) {
				return false, e
			}
			continue
		}
		if entry.IsDir() {
			if entry.Type()&(fs.ModeSymlink|fs.ModeIrregular) != 0 {
				continue
			}
			switch entry.Name() {
			case ".git", "node_modules", "vendor", ".next", ".venv", "dist", "build", "coverage", "bin", "obj":
				continue
			}
			if excluded(name) {
				continue
			}
			if _, e := r.Lstat(path.Join(name, ".git")); e == nil {
				continue
			} else if !errors.Is(e, fs.ErrNotExist) {
				return false, e
			}
			found, e := scanLegacy(r, name)
			if e != nil || found {
				return found, e
			}
		}
	}
	return false, nil
}
