# Archbase schemas

The public contracts live in `schemas/` and use JSON Schema Draft 2020-12. YAML documents are converted to their JSON data model before validation.

## Versioning

Every document must declare `schemaVersion: 1`. The workspace index is a new document contract; the legacy single-scope schema is retained as a data definition but is no longer discovered by workspace commands.

- Removing or renaming a field, changing its meaning, or tightening a previously valid value requires a new `schemaVersion`.
- A future release may add optional fields without changing the schema version.
- Unknown core fields are rejected. Implementations may store experimental data only in the free-form `metadata` object.

## IDs and versions

Pattern IDs use `stack/type@id`, for example `next/page@1234`. All ID segments are lowercase and may contain digits, dots, underscores, and hyphens. Versions use semantic versioning.

## Contracts

- `manifest.schema.json` supports a single-file `pattern` and a multi-file `pattern-bundle`. Source and destination paths must be relative and remain inside the pattern root.
- `workspace.schema.json` defines the central `.archbase/scopes.yaml` index, a sorted, unique array of scope documents with directory paths relative to the project root.
- `scope.schema.json` describes an applicable path, its local or registry pattern, optional origin, and deterministic behavior. Local patterns require `pattern.root`.
- `rule.schema.json` is the agent-neutral initial architecture rule model: metadata, path-to-pattern associations, and textual restrictions. Exporter-specific fields do not belong in this contract.

Schemas are strict at every core object level. Only `metadata` accepts arbitrary extension keys.

After schema validation, the pattern loader performs semantic validation that cannot be expressed by the document alone: source and destination uniqueness, root confinement, symlink rejection, regular-file checks, and required-file existence. Optional files remain represented in the loaded bundle with `present: false`.

Central scopes use project-relative `scope.path` values (`.` for the root), relative roots confined below the central `.archbase`, and `nearestScopeWins: true`. The longest applicable directory prefix wins; prefix matches respect path segment boundaries. The complete index is validated before selection. The selected bundle is revalidated; invalid scopes never fall back to ancestors.

Rule scope paths are normalized slash-separated relative globs. Absolute paths, traversal, backslashes, duplicate path-to-pattern associations, and whitespace-only restrictions are rejected by semantic validation after schema validation.
Manifest file entries accept optional `format: raw|markdown-code` (default `raw`). A markdown-code source ends in `.md` and wraps the logical file in exactly one typed code fence. See [workspace.md](workspace.md) for aliases, validation and newline normalization.
