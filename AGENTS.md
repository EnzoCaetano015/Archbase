# Archbase contribution rules

## Product boundaries

- Patterns define how a type of code is structured; they are not feature boilerplates.
- Rules define architecture, paths, and dependency constraints; they do not duplicate pattern source code.
- The nearest `.archbase` scope will win when scope resolution is implemented.
- Local customization will take precedence over registry content.

## Current milestone boundaries

- The current milestone implements TASK-001 through TASK-008.
- Keep Go packages under `internal/` until a public Go API is intentionally designed.
- Public Git registries may be cloned by the registry core; authentication remains out of scope.
- Do not add pattern installation, local pattern creation, scope resolution, rule exporters, or MCP in this milestone.
- Existing files must never be overwritten unless the caller explicitly opts in.
- Changes to public YAML contracts require matching schema, tests, and documentation updates.
- Registry entries must remain sorted by ID and every declared required file must pass bundle validation.
- When TASK-010 is implemented, names such as `pages-standard` map to canonical local IDs such as `local/pages-standard@1`.
