# Archbase canonical rules

Patterns define how a type of code is structured. Rules define where those types belong, how layers relate, and which pattern applies to each path. A rule references pattern IDs and never copies their example source code.

## Canonical contract

A schema version 1 rule contains an ID, name, optional description, semantic version, one or more path-to-pattern associations, textual architectural restrictions, and optional metadata. The contract deliberately contains no Cursor, GitHub Copilot, Claude, or AGENTS-specific fields.

Rule paths are relative slash-separated globs such as `src/pages/**`. Paths must be normalized and remain inside the destination project. Every restriction must contain meaningful text, and duplicate path-to-pattern associations are invalid.

## Official rules

- `architecture/astro-content-site@1` separates Astro file routes, document layouts, server-rendered components, validated content collections, and pure utilities.
- `architecture/discordjs-music-bot@1` organizes CommonJS Discord bots into command modules, a client composition root, explicit command publication, and player events. See the [Ribby Bot reference notes](../registry/rules/architecture/discordjs-music-bot/1/README.md).
- `architecture/dotnet-multiproject@1` separates ASP.NET Core composition, application contracts and services, persistence, shared contracts, global failures, and scheduled jobs across projects.
- `architecture/nestjs-prisma-modular@1` separates NestJS feature modules, validated transport contracts, authentication guards, external HTTP adapters, global response policies, and Prisma infrastructure.
- `architecture/next-feature-app-router@1` separates App Router pages, layouts, and HTTP handlers from feature-owned actions, application services, components, persistence, models, and schemas.
- `architecture/next-modular@1` separates pages, components, hooks, and utilities and associates each path with the corresponding Next pattern.
- `architecture/dotnet-layered@1` defines Controller → Service → Repository responsibilities and prevents reverse dependencies.
- `architecture/python-fastapi-modular@1` separates FastAPI transport, Pydantic contracts, application orchestration, and PostgreSQL persistence inside feature modules.
- `architecture/react-tailwind-modular@1` separates Tailwind pages, shadcn-style primitives, domain components, endpoint paths, typed query controllers, and a page registry.
- `architecture/react-vite-modular@1` separates React pages, reusable UI, typed API access, contracts, shared hooks, utilities, and centralized routes.
- `architecture/spring-boot-layered@1` separates Spring MVC controllers, transactional services, JPA repositories, audited entities, validated records, MapStruct mappers, JWT security, and global response policies.
- `architecture/spring-boot-reactive-mongo@1` follows the Horizon Java API structure with reactive MongoDB repositories, service interfaces and implementations, and separate request and response contracts. See the [reference notes](../registry/rules/architecture/spring-boot-reactive-mongo/1/README.md) for scope and adaptations.

The official catalog is embedded and works offline. Directory and public Git registries may provide their own `rules/index.yaml`; configured sources are ordered and the first source containing an ID wins. A resolved rule is usable only when all referenced pattern IDs resolve through the configured pattern sources.

## CLI

```bash
arc rules list
arc rules inspect architecture/next-modular@1
arc rules add architecture/next-modular@1 --format agents --destination .
arc rules add architecture/next-modular@1 --format claude --destination .
arc rules add architecture/next-modular@1 --format cursor --destination .
arc rules add architecture/next-modular@1 --format copilot --destination .
arc remove architecture/next-modular@1 --destination .
arc remove --all --destination .
```

The destination defaults to the invocation directory. Global Git registry flags apply to rules and patterns together: Git precedes embedded, only not-found permits fallback, and stale-cache warnings go to stderr. Removal is entirely offline.

## One document, four native entry points

Each RuleID is stored once in `.archbase/rules/<normalized-id>.md`. For example, `architecture/next-modular@1` becomes `architecture-next-modular-1.md`. Its frontmatter contains `id`, `name`, `description`, `version`, `source`, and `scopes`; the body contains the complete architecture, pattern references, restrictions, and inspection commands, never pattern example code. An Archbase ownership marker carries the original RuleID. Canonical YAML schemas are unchanged.

| Format | Entry point | Link prefix |
| --- | --- | --- |
| `agents` | `AGENTS.md` | `.archbase/rules/` |
| `claude` | `CLAUDE.md` | `.archbase/rules/` |
| `copilot` | `.github/copilot-instructions.md` | `../.archbase/rules/` |
| `cursor` | `.cursor/rules/archbase.mdc` | `../../.archbase/rules/` |

Native locations follow the [Claude memory documentation](https://code.claude.com/docs/en/memory), [GitHub custom instructions](https://docs.github.com/en/copilot/concepts/prompting-and-context/custom-instructions?tool=vscode), and [Cursor rules](https://cursor.com/docs/rules).

Each entry point has one managed index between these exact markers:

```markdown
<!-- archbase:rules start -->
...
<!-- archbase:rules end -->
```

Entries are sorted by RuleID, with an objective description (falling back to the name), project-relative scope globs, and a relative document link. Markdown metacharacters are escaped. An explicit instruction tells the agent to open every applicable document before editing; a Markdown link is not an automatic import.

Selecting a format creates its entry point or appends its index. Every already activated format is synchronized on additions, updates, and removals. A format is activated only by a complete managed index, not merely by the existence of its file. Activating another format includes all installed rules; other formats are not created automatically. The document remains shared.

New Cursor entry points have `description` and `alwaysApply: true` frontmatter. Existing Cursor frontmatter must be valid YAML with `alwaysApply: true`; otherwise installation fails, even with overwrite. Compatible frontmatter and all external content remain byte-for-byte unchanged, including CRLF.

## Updates, conflicts, and cleanup

Identical documents are reused without rewriting them. A changed document requires `--overwrite` and must belong to the same RuleID. Foreign files and filename collisions between different IDs are never overwritten. Index updates automatically touch only the managed block; `--merge` is accepted in all formats solely as an obsolete, no-op compatibility flag.

A first rule installation also initializes an empty `.archbase/scopes.yaml` in the same transaction, so patterns can be installed later.

Reapplying a rule removes its legacy blocks from `AGENTS.md` and `CLAUDE.md`, including the root, and its recognized individual Cursor/Copilot exports. Other rules and external instructions remain intact. A nested instruction file is deleted only if it consisted solely of the removed blocks and whitespace. No new per-scope instruction files are created, and no migration command is required.

## Removal and safety

`arc remove <id>` removes the central document, matching legacy exports, and references in every active index. `--all` removes all managed rules, preserving patterns, scopes, and foreign files. Removing the final rule strips the index blocks but keeps all four native entry files, even when empty (or containing only Cursor frontmatter). A later installation activates only the selected format because an empty entry file has no managed block.

Removal validates canonical filenames and ownership IDs; links in Markdown never authorize deletion. Older markerless exports are recognized only by their strict generated fingerprint. Malformed, duplicated, reversed, or overlapping markers abort the operation before writes. Legacy discovery skips `.archbase`, dependencies, build output, and nested Git projects.

Paths are confined to the destination and reject symlinks. Patterns and rules share a contextual workspace lock. Document, workspace initialization, indices, and legacy cleanup are prevalidated and applied transactionally: a failure restores changed/deleted files and removes newly created files/directories. Operational outputs list only paths actually changed or removed. A successful no-op does not rewrite files.

The six read-only MCP tools retain their existing semantics; local instruction indices do not change registry-based rule resolution.
