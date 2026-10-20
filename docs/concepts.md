# Archbase concepts

Archbase makes project conventions inspectable by developers and AI coding agents. Patterns describe code structure, rules describe architecture, and scopes select the pattern that applies to a project path.

## Patterns

A pattern is a structural example for one type of code, such as a page, controller, or repository. It does not represent a business feature or generate an entire application.

Patterns use canonical IDs such as `next/page@1234`, `python/router@2784`, and `dotnet-multiproject/controller@2947`. Each bundle includes a manifest and declared source files, validated before use.

```bash
arc add next/page@1234 .
arc inspect next/page@1234
```

The first command stores the validated bundle in a project scope. The second inspects the registry pattern. See the [registry guide](registry.md) for the catalog and source resolution.

## Architecture rules

Rules define where code belongs, layer responsibilities, and allowed dependency directions. They associate relative paths such as `src/pages/**` with pattern IDs without copying pattern source examples.

The same canonical rule can be exported to Cursor, GitHub Copilot, or `AGENTS.md`. See the [rules guide](rules.md) for commands, formats, and conflict handling.

## Scopes and resolution

A project stores every scope in the single root `.archbase/scopes.yaml`. Scopes associate project-relative directory paths with patterns; the directories themselves need no metadata folders.

```text
project/
├── .archbase/
│   ├── scopes.yaml
│   └── patterns/<scope-hash>/<pattern-hash>/
├── src/pages/
└── src/pages/admin/
```

Run installation commands from the project root:

```bash
arc add next/page@1234 src/pages
arc create pages-standard src/pages/admin --from next/page@1234
arc resolve src/pages/Home.tsx
arc resolve src/pages/admin/Dashboard.tsx
```

The most specific directory scope wins, respecting path segment boundaries. An invalid selected scope reports an error instead of falling back to a parent. The target file does not need to exist. Each scope has one active pattern; activating another preserves stored bundles and refuses collisions.

## Local customization

Each scope owns its pattern copy, even when another scope uses the same PatternID. Local names become canonical IDs such as `local/pages-standard@1`. The origin retains the registry source, original ID, and version.

Use `arc resolve src/pages/admin/Dashboard.tsx` to find the printed pattern root. Edit the code inside `Example/Example.tsx.md`, preserving the typed Markdown fence. Examples are documents rather than compiler inputs; the loader and MCP expose extracted code without backticks. Local customization takes precedence over registry content and is revalidated on every resolution.

```bash
arc inspect src/pages/admin/Dashboard.tsx
```

See [workspace layout and document format](workspace.md), [Getting started](getting-started.md), and the [MCP server](mcp.md).
