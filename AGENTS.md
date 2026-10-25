# Archbase contribution rules

## Product boundaries

- Patterns define how a type of code is structured; they are not feature boilerplates.
- Rules define architecture, paths, and dependency constraints; they do not duplicate pattern source code.
- The most specific directory scope in the root `.archbase/scopes.yaml` wins; invalid scopes must not be hidden by ancestors.
- Local customization will take precedence over registry content.

## Current milestone boundaries

- TASK-001 through TASK-023 are implemented; the first public milestone is complete.
- Keep Go packages under `internal/` until a public Go API is intentionally designed.
- Public Git registries may be cloned by the registry core; authentication remains out of scope.
- Stable releases are triggered only by existing `vMAJOR.MINOR.PATCH` tags and must pass the complete verification gate before publication.
- Unsigned release archives and checksums must remain byte-for-byte reproducible for identical source, version, and timestamp inputs.
- Linux is the official distribution channel. macOS and Windows artifacts and installers remain available only as unsigned previews.
- Release-signing preparation remains in scope, but the current SignPath Foundation application was not approved. Reapply only after the project has meaningful public adoption and external recognition.
- If a trusted signing workflow is approved in the future, signed executables may differ byte-for-byte because trusted Authenticode timestamps are intentionally non-deterministic.
- Public checksum manifests must always describe the final artifacts delivered to users, including signed Windows archives after signing is activated.
- The SignPath integration must remain inactive until a future application is approved; do not add attribution, credentials, secret placeholders, or claim an affiliation or signed release before signature validation succeeds.
- Registry authentication, package-manager distribution, attestations, and non-stdio MCP transports remain out of scope.
- Existing files must never be overwritten unless the caller explicitly opts in.
- Self-update is explicit through arc update only: use official stable GitHub releases over HTTPS, verify SHA-256 and the downloaded executable version, and replace only the invoked installation. Never downgrade, elevate automatically, modify projects/PATH, or introduce background update checks. Preserve recovery copies when rollback fails.
- Changes to public YAML contracts require matching schema, tests, and documentation updates.
- Registry entries must remain sorted by ID and every declared required file must pass bundle validation.
- Local names such as `pages-standard` map to canonical IDs such as `local/pages-standard@1`.
- A scope has one active pattern; activating another preserves stored pattern directories and never overwrites a collision.
- New add/create commands use the invocation directory as project root unless --project-root explicitly overrides it; do not detect a Git root automatically.
- Store scopes only in the root .archbase/scopes.yaml and independent bundles under patterns/<readable-scope-name>/<readable-pattern-name>; keep old hashed roots readable and never overwrite stored bundles.
- Store textual examples as typed Markdown documents (for example file.ts.md); consumers receive extracted code while installation preserves the Markdown document.
- Legacy per-directory scopes have no runtime compatibility or migration command in this release.
- Local pattern roots remain confined to `.archbase` and are fully revalidated during resolution.
- Canonical rules remain agent-neutral and reference patterns by ID instead of copying their source examples.
- Rule registries use `rules/index.yaml`; entries remain sorted and every referenced rule document must validate.
- Rules are stored once in `.archbase/rules`; AGENTS, Claude, Cursor, and Copilot use synchronized managed indices in their native entry files. Activate only the selected format and synchronize every already active format.
- Identical owned rule documents are reused; changing one requires explicit overwrite for the same RuleID. Never overwrite foreign files or slug collisions. Merge is a deprecated no-op; index updates preserve all external bytes.
- Cursor index frontmatter must be valid with alwaysApply true; incompatible external frontmatter is a conflict.
- Every rule export path remains confined to its destination and multi-file exports must roll back on failure.
- Rule removal is offline and may delete only owned central documents and recognized legacy exports/blocks. Synchronize active indices, preserve external bytes and the native entry files, and never infer deletion authority from Markdown links.
- Pattern and rule writes share the project workspace lock with cancellation; document, indices, scopes initialization, and legacy cleanup must roll back together.
- MCP tools are read-only, use stdio, and must keep protocol output isolated on stdout.
- MCP project paths remain confined to the configured project root and must reject traversal and symlinks.
