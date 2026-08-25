# Archbase contribution rules

## Product boundaries

- Patterns define how a type of code is structured; they are not feature boilerplates.
- Rules define architecture, paths, and dependency constraints; they do not duplicate pattern source code.
- The nearest `.archbase` scope wins; invalid nearest scopes must not be hidden by ancestors.
- Local customization will take precedence over registry content.

## Current milestone boundaries

- The first milestone implements TASK-001 through TASK-013.
- Keep Go packages under `internal/` until a public Go API is intentionally designed.
- Public Git registries may be cloned by the registry core; authentication remains out of scope.
- Rule exporters, MCP, releases, and binary installation remain outside the completed first milestone.
- Existing files must never be overwritten unless the caller explicitly opts in.
- Changes to public YAML contracts require matching schema, tests, and documentation updates.
- Registry entries must remain sorted by ID and every declared required file must pass bundle validation.
- Local names such as `pages-standard` map to canonical IDs such as `local/pages-standard@1`.
- A scope has one active pattern; activating another preserves stored pattern directories and never overwrites a collision.
- Local pattern roots remain confined to `.archbase` and are fully revalidated during resolution.
