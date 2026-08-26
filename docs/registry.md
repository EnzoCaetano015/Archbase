# Archbase registry core

The registry core resolves IDs in `stack/type@id` form without depending on CLI commands. A resolver receives an ordered list of sources; the first source containing the requested ID wins. A malformed registry is a hard error and is never hidden by falling back to a later source.

## Official patterns

The binary embeds the initial catalog:

- `next/page@1234`
- `next/component@4821`
- `next/hook@9214`
- `next/util@3378`
- `dotnet/controller@7743`
- `dotnet/repository@5532`
- `dotnet/service@1172`

Patterns describe code structure rather than application features. Registry entries are sorted by ID and validated against their manifests and declared files when a source is opened.

The same registry may include an independent `rules/index.yaml` catalog. Rule entries point to directories containing `rule.yaml`; IDs, versions, ordering, path confinement, symlinks, and the complete canonical document are validated. A registry without a `rules/` directory remains a valid pattern-only registry.

## Git sources

`GitSourceConfig` is an internal API. It accepts a public `https`, `git`, or absolute `file` URL, a branch or tag, an optional registry subdirectory, an absolute cache root, and an optional TTL. The CLI exposes the same choices through global `--registry-*` flags.

- URLs containing credentials, insecure HTTP, and SSH are rejected.
- The default TTL is 15 minutes.
- Cache directories use a SHA-256 key derived from URL and ref, so remote details are not exposed in path names.
- Clone promotion is atomic and fetch/reset updates happen under a contextual cross-process lock.
- If refresh fails, cached content is used only after complete registry and bundle validation. The result is marked stale and includes a warning.
- An invalid cache or a canceled context produces an error.

Patterns and rules use the same contextual Git checkout, cache key, TTL, lock, and stale fallback. Each catalog validates its own cached content before using a stale snapshot.

Environment variables, private-registry authentication, and user-facing cache management remain outside the first milestone.

## Local installation and identity

`arc add` copies only the validated manifest and declared bundle files into `.archbase/patterns/<type>-<id>`. Registry README files and undeclared extras are not installed.

`arc create pages-standard <scope>` uses the canonical ID `local/pages-standard@1`. It starts from a minimal valid example unless `--from <pattern-id>` derives it from a registry pattern. Derived manifests receive the local identity while `scope.yaml` records the original registry, ID, and version.

One pattern is active per scope. Activating a new pattern preserves previously stored patterns, refuses a directory collision, and rolls back the new directory if the scope update fails.
