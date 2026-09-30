# Archbase contribution rules

## Product boundaries

- Patterns define how a type of code is structured; they are not feature boilerplates.
- Rules define architecture, paths, and dependency constraints; they do not duplicate pattern source code.
- The nearest `.archbase` scope wins; invalid nearest scopes must not be hidden by ancestors.
- Local customization will take precedence over registry content.

## Current milestone boundaries

- TASK-001 through TASK-023 are implemented; the first public milestone is complete.
- Keep Go packages under `internal/` until a public Go API is intentionally designed.
- Public Git registries may be cloned by the registry core; authentication remains out of scope.
- Stable releases are triggered only by existing `vMAJOR.MINOR.PATCH` tags and must pass the complete verification gate before publication.
- Unsigned release archives and checksums must remain byte-for-byte reproducible for identical source, version, and timestamp inputs.
- Release-signing preparation and Authenticode signing through the approved SignPath workflow are in scope. Signed executables may differ byte-for-byte because trusted Authenticode timestamps are intentionally non-deterministic.
- Public checksum manifests must always describe the final artifacts delivered to users, including signed Windows archives after signing is activated.
- The SignPath integration must remain inactive until the project is approved; do not add credentials, secret placeholders, or claim that a release is signed before signature validation succeeds.
- Registry authentication, package-manager distribution, attestations, and non-stdio MCP transports remain out of scope.
- Existing files must never be overwritten unless the caller explicitly opts in.
- Changes to public YAML contracts require matching schema, tests, and documentation updates.
- Registry entries must remain sorted by ID and every declared required file must pass bundle validation.
- Local names such as `pages-standard` map to canonical IDs such as `local/pages-standard@1`.
- A scope has one active pattern; activating another preserves stored pattern directories and never overwrites a collision.
- Local pattern roots remain confined to `.archbase` and are fully revalidated during resolution.
- Canonical rules remain agent-neutral and reference patterns by ID instead of copying their source examples.
- Rule registries use `rules/index.yaml`; entries remain sorted and every referenced rule document must validate.
- Cursor and Copilot rule exports require explicit overwrite on conflict; AGENTS exports update only RuleID-specific managed blocks with explicit merge.
- Every rule export path remains confined to its destination and multi-file exports must roll back on failure.
- MCP tools are read-only, use stdio, and must keep protocol output isolated on stdout.
- MCP project paths remain confined to the configured project root and must reject traversal and symlinks.
