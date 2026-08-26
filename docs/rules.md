# Archbase canonical rules

Patterns define how a type of code is structured. Rules define where those types belong, how layers relate, and which pattern applies to each path. A rule references pattern IDs and never copies their example source code.

## Canonical contract

A schema version 1 rule contains an ID, name, optional description, semantic version, one or more path-to-pattern associations, textual architectural restrictions, and optional metadata. The contract deliberately contains no Cursor, GitHub Copilot, or AGENTS-specific fields.

Rule paths are relative slash-separated globs such as `src/pages/**`. Paths must be normalized and remain inside the destination project. Every restriction must contain meaningful text, and duplicate path-to-pattern associations are invalid.

## Official rules

- `architecture/next-modular@1` separates pages, components, hooks, and utilities and associates each path with the corresponding Next pattern.
- `architecture/dotnet-layered@1` defines Controller → Service → Repository responsibilities and prevents reverse dependencies.

The official catalog is embedded and works offline. Directory and public Git registries may provide their own `rules/index.yaml`; configured sources are ordered and the first source containing an ID wins. A resolved rule is usable only when all referenced pattern IDs resolve through the configured pattern sources.

Rule listing, inspection, and exporters remain planned for TASK-016 through TASK-019.
