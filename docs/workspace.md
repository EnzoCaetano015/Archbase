# Central workspace

Run `arc add` and `arc create` from the project root. They use the invocation directory, without Git-root detection. Pass `--project-root /path/to/project` to select a different root; CLI positional paths remain relative to the invocation directory. The terminal UI resolves paths relative to its displayed project directory. MCP inputs are relative to `mcp serve --project-root` (default `.`).

```text
project/
├── .archbase/
│   ├── scopes.yaml
│   └── patterns/
│       └── <scope-key>/
│           └── <pattern-key>/
│               ├── manifest.yaml
│               └── application.ts.md
└── features/
    └── storefront/
        ├── application/
        └── components/
```

`scope-key` is the lowercase full SHA-256 of the normalized project-relative scope path (`.` at the root); `pattern-key` is the lowercase full SHA-256 of its canonical PatternID. Identity and paths remain readable in `scopes.yaml` and `arc resolve`/`arc inspect` output. Use the printed pattern root to find files; do not guess hash paths.

The index has `schemaVersion: 1`, optional `metadata`, and a `scopes` array. Each entry uses the scope contract: `schemaVersion: 1`, `scope.path`, `pattern`, `origin` when applicable, `behavior`, and optional `metadata`. Entries are sorted lexicographically by path. `scope.path` is `.` or a normalized relative directory without globs. Local `pattern.root` is relative to the central `.archbase`.

The most specific ancestral directory wins: `src/pages/admin` overrides `src/pages`, and `src/pages` does not match `src/pages-old`. Targets may be files, directories, or paths not created yet. Without an explicit root, read commands find the nearest ancestor containing `.archbase` and then consult its index. The full index must be valid. An invalid selected bundle raises an error; it is never hidden by a parent scope.

Each scope keeps an independent copy of a pattern. Editing one scope's Markdown example does not change another scope. Activating a new pattern preserves previously stored bundles; installing the same ID again in the same scope reports a collision. Only the manifest and declared sources are installed, without registry READMEs.

Writes are serialized by an exclusive `.archbase.lock` file at the project root and staged before promotion. Waiting honors context cancellation. Normal completion or failure removes the lock and staging files. After a process crash, verify that no writer is running before manually removing a leftover lock. Failed index updates roll back the newly promoted bundle and preserve the previous index.

This release does not load the old per-directory `scope.yaml` layout and does not provide a migration command. Existing projects must be set up again using the central workspace. Preserve customized legacy files before replacing them.

## Typed Markdown examples

An example such as `application.ts.md` is a Markdown document containing exactly one typed code fence. Its manifest declares:

```yaml
- source: application.ts.md
  destination: "{{Name}}/{{Name}}.ts"
  required: true
  format: markdown-code
```

````markdown
```typescript
export function example(): string {
  return "example"
}
```
````

Edit the code inside the fence and preserve its language and delimiters. The document must be UTF-8 and contain no non-whitespace text outside the fence. Longer backtick fences allow backticks inside examples. Exactly matching standalone delimiters close the block. Unterminated blocks, multiple blocks, and incompatible languages are errors with manifest, field, and file context.

| Logical extension | Language | Accepted aliases |
| --- | --- | --- |
| `.ts` | `typescript` | `ts` |
| `.tsx` | `tsx` | — |
| `.js`, `.mjs`, `.cjs` | `javascript` | `js` |
| `.jsx` | `jsx` | — |
| `.cs` | `csharp` | `cs` |
| `.py` | `python` | — |
| `.java` | `java` | — |
| `.astro` | `astro` | — |
| `.yaml`, `.yml` | `yaml` | `yml` |
| `.json` | `json` | — |
| `.md` | `markdown` | `md` |
| `.go`, `.html`, `.css`, `.sql` | extension without dot | — |
| `.sh` | `bash` | `sh` |
| other | `text` | `plaintext` |

The final `.md` is storage-only: `.ts.md` identifies TypeScript; the destination retains `.ts`. The loader keeps both original document bytes and extracted code. MCP `get_pattern_files` returns the code and `language`, without the delimiters. Installation copies the original document.

Raw registry files remain supported with omitted `format` or `format: raw`. Installation converts UTF-8 text without NUL bytes into typed Markdown. Binary sources remain raw and MCP returns base64. The minimal local pattern uses `Example.txt.md` with language `text`.

During conversion, CRLF becomes LF and trailing newlines are normalized to one final newline. Interior content remains intact. Existing Markdown documents retain their stored bytes; reading extracts LF code with its final newline. Keep manifests, indexes, and READMEs in their native formats.
