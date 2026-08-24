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

## Git sources

`GitSourceConfig` is an internal API in the current milestone. It accepts a public `https`, `git`, or absolute `file` URL, a branch or tag, an optional registry subdirectory, an absolute cache root, and an optional TTL.

- URLs containing credentials, insecure HTTP, and SSH are rejected.
- The default TTL is 15 minutes.
- Cache directories use a SHA-256 key derived from URL and ref, so remote details are not exposed in path names.
- Clone promotion is atomic and fetch/reset updates happen under a contextual cross-process lock.
- If refresh fails, cached content is used only after complete registry and bundle validation. The result is marked stale and includes a warning.
- An invalid cache or a canceled context produces an error.

Flags, environment variables, private-registry authentication, and user-facing cache management remain outside TASK-005 through TASK-008.

## Future local identity

When `arc create` is implemented, a local name such as `pages-standard` will use the canonical ID `local/pages-standard@1` and start from a minimal valid example unless it derives from a registry pattern.
