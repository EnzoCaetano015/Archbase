# Archbase MCP server

Archbase exposes its validated pattern, scope, and rule core to AI agents through the Model Context Protocol. The first server uses stdio and starts from the same `arc` binary:

```bash
arc mcp serve --project-root .
```

The global `--registry-*` flags apply to MCP exactly as they do to the other commands. A configured public Git registry has precedence over the embedded catalog, and a valid stale cache is reported through tool output warnings.

## Project boundary

`--project-root` defaults to the current directory and must identify an existing regular directory. Relative tool paths are resolved from this root. Absolute paths are accepted only when they remain inside it. Traversal and symlinks in existing path components are rejected.

All tools are read-only. They may refresh the existing Git registry cache, but they never install patterns, change scopes, export rules, or write generated code. Stdout is reserved for MCP protocol messages.

## Tools

- `search_patterns` searches ID, name, and description case-insensitively and returns deterministic pattern summaries.
- `get_pattern` returns the validated manifest for a registry PatternID.
- `resolve_pattern` resolves the nearest `.archbase` scope for a project path.
- `get_pattern_files` accepts exactly one of `patternId` or `path` and returns every declared bundle file. UTF-8 content is returned as text; other bytes use base64.
- `get_scope_rules` returns registry rules whose pattern association and glob match the resolved project path. A local pattern's recorded origin is considered.
- `list_project_scopes` validates and lists every scope below the configured project root.

Normal results use typed structured content and include `stale` and `warnings` where registry access is involved. Invalid arguments, missing IDs, missing scopes, corrupt bundles, and paths outside the project are returned as visible MCP tool errors so an agent can correct its request.
