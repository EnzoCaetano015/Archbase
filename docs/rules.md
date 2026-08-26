# Archbase canonical rules

Patterns define how a type of code is structured. Rules define where those types belong, how layers relate, and which pattern applies to each path. A rule references pattern IDs and never copies their example source code.

## Canonical contract

A schema version 1 rule contains an ID, name, optional description, semantic version, one or more path-to-pattern associations, textual architectural restrictions, and optional metadata. The contract deliberately contains no Cursor, GitHub Copilot, or AGENTS-specific fields.

Rule paths are relative slash-separated globs such as `src/pages/**`. Paths must be normalized and remain inside the destination project. Every restriction must contain meaningful text, and duplicate path-to-pattern associations are invalid.

## Official rules

- `architecture/next-modular@1` separates pages, components, hooks, and utilities and associates each path with the corresponding Next pattern.
- `architecture/dotnet-layered@1` defines Controller → Service → Repository responsibilities and prevents reverse dependencies.

The official catalog is embedded and works offline. Directory and public Git registries may provide their own `rules/index.yaml`; configured sources are ordered and the first source containing an ID wins. A resolved rule is usable only when all referenced pattern IDs resolve through the configured pattern sources.

## CLI

- `arc rules list` lists IDs, versions, sources, and descriptions deterministically.
- `arc rules inspect <rule-id>` displays identity, source, scopes, pattern IDs, and restrictions.
- `arc rules add <rule-id> --format cursor|copilot|agents` exports a resolved rule. `--destination` defaults to the current directory.

The global Git registry flags apply to rules and patterns together. A configured Git source is tried before the embedded catalog; only a missing ID permits fallback. Valid stale-cache warnings are printed to stderr.

## Export formats and conflicts

The full RuleID becomes a stable filename component: `architecture/next-modular@1` becomes `architecture-next-modular-1`.

- Cursor writes `.cursor/rules/<normalized-id>.mdc` with `description`, `globs`, and `alwaysApply: false` frontmatter.
- Copilot writes `.github/instructions/<normalized-id>.instructions.md` with `applyTo` frontmatter.
- AGENTS writes `AGENTS.md` at every static scope prefix, grouping scopes with the same directory. Each generated section is bounded by RuleID-specific Archbase markers.

Cursor and Copilot refuse existing files unless `--overwrite` is explicit. AGENTS rejects `--overwrite`; an existing file requires `--merge`. Merge replaces one complete matching managed block or appends a missing block while preserving external content. Incomplete, reversed, or duplicate markers are errors.

Every generated path is confined to the destination, and symlinked paths are rejected. Multi-file exports are fully prevalidated and rolled back if any write fails. Exported content contains architecture, restrictions, pattern IDs, and commands for inspecting patterns; it never copies pattern example source code.
