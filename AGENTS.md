# Archbase contribution rules

## Product boundaries

- Patterns define how a type of code is structured; they are not feature boilerplates.
- Rules define architecture, paths, and dependency constraints; they do not duplicate pattern source code.
- The nearest `.archbase` scope wins; invalid nearest scopes must not be hidden by ancestors.
- Local customization will take precedence over registry content.

## Current milestone boundaries

- TASK-001 through TASK-021 are implemented.
- Keep Go packages under `internal/` until a public Go API is intentionally designed.
- Public Git registries may be cloned by the registry core; authentication remains out of scope.
- Releases, binary installation, and the complete public first-flow guide remain outside the current milestone.
- Existing files must never be overwritten unless the caller explicitly opts in.
- Changes to public YAML contracts require matching schema, tests, and documentation updates.
- Registry entries must remain sorted by ID and every declared required file must pass bundle validation.
- Local names such as `pages-standard` map to canonical IDs such as `local/pages-standard@1`.
- A scope has one active pattern; activating another preserves stored pattern directories and never overwrites a collision.
- Local pattern roots remain confined to `.archbase` and are fully revalidated during resolution.
- Canonical rules remain agent-neutral and reference patterns by ID instead of copying their source examples.
- Rule registries use `rules/index.yaml`; entries remain sorted and every referenced rule document must validate.
- Cursor and Copilot rule exports require explicit overwrite on conflict; AGENTS exports update only RuleID-specific managed blocks with explicit merge.
- Every rule export path remains confined to its destination and multi-file exports must roll back on failure.
- MCP tools are read-only, use stdio, and must keep protocol output isolated on stdout.
- MCP project paths remain confined to the configured project root and must reject traversal and symlinks.
