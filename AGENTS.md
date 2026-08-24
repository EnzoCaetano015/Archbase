# Archbase contribution rules

## Product boundaries

- Patterns define how a type of code is structured; they are not feature boilerplates.
- Rules define architecture, paths, and dependency constraints; they do not duplicate pattern source code.
- The nearest `.archbase` scope will win when scope resolution is implemented.
- Local customization will take precedence over registry content.

## Foundation boundaries

- The current milestone implements TASK-001 through TASK-004 only.
- Keep Go packages under `internal/` until a public Go API is intentionally designed.
- Do not add network registry fetching, pattern installation, scope resolution, rule exporters, or MCP in this milestone.
- Existing files must never be overwritten unless the caller explicitly opts in.
- Changes to public YAML contracts require matching schema, tests, and documentation updates.
