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

A scope stores its configuration in `.archbase/scope.yaml`. A project can have nested scopes:

```text
project/
├── .archbase/
│   └── scope.yaml
└── src/
    └── pages/
        └── admin/
            └── .archbase/
                └── scope.yaml
```

```bash
arc resolve src/pages/Home.tsx
arc resolve src/pages/admin/Dashboard.tsx
```

Resolution walks from the requested path toward its ancestors. The nearest `.archbase` scope wins. If that scope is invalid, resolution reports an error instead of falling back to an ancestor. The target file does not need to exist.

Each scope has one active pattern. Activating another preserves stored pattern directories and refuses directory collisions.

## Local customization

Create a project-owned pattern from a registry pattern:

```bash
arc create pages-standard ./src/pages --from next/page@1234
```

The local name becomes `local/pages-standard@1`. Its files can be edited inside the scope:

```text
src/pages/.archbase/
├── scope.yaml
└── patterns/
    └── pages-standard/
        └── Example/
            ├── Example.tsx
            ├── Example.hook.ts
            └── Example.utils.ts
```

Local customization takes precedence over registry content. The scope retains the original pattern identity as its origin, while resolution uses the local files. Local roots remain confined to `.archbase` and are revalidated during resolution.

To inspect the active local pattern, pass a project path:

```bash
arc inspect src/pages/Home.tsx
```

For a complete example, follow [Getting started](getting-started.md). Compatible AI agents can also inspect this context through the [MCP server](mcp.md).
