# Archbase schemas

The public contracts live in `schemas/` and use JSON Schema Draft 2020-12. YAML documents are converted to their JSON data model before validation.

## Versioning

Every document must declare `schemaVersion: 1`.

- Removing or renaming a field, changing its meaning, or tightening a previously valid value requires a new `schemaVersion`.
- A future release may add optional fields without changing the schema version.
- Unknown core fields are rejected. Implementations may store experimental data only in the free-form `metadata` object.

## IDs and versions

Pattern IDs use `stack/type@id`, for example `next/page@1234`. All ID segments are lowercase and may contain digits, dots, underscores, and hyphens. Versions use semantic versioning.

## Contracts

- `manifest.schema.json` supports a single-file `pattern` and a multi-file `pattern-bundle`. Source and destination paths must be relative and remain inside the pattern root.
- `scope.schema.json` describes an applicable path, its local or registry pattern, optional origin, and deterministic behavior. Local patterns require `pattern.root`.
- `rule.schema.json` is the agent-neutral initial architecture rule model: metadata, path-to-pattern associations, and textual restrictions. Exporter-specific fields do not belong in this contract.

Schemas are strict at every core object level. Only `metadata` accepts arbitrary extension keys.

After schema validation, the pattern loader performs semantic validation that cannot be expressed by the document alone: source and destination uniqueness, root confinement, symlink rejection, regular-file checks, and required-file existence. Optional files remain represented in the loaded bundle with `present: false`.

Generated local scopes use `scope.path: "."`, relative roots below `.archbase`, and `nearestScopeWins: true`. Resolution begins at the requested file or directory and walks toward the filesystem root. The first `.archbase` directory wins; an invalid nearest scope is reported instead of being hidden by an ancestor.
