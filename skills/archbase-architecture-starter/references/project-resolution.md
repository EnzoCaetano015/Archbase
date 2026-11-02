# Project and scope resolution

The project root is the explicitly supplied root or invocation directory, not an
automatically detected Git root. Read only paths inside it; reject traversal and
symlinks. In a monorepo, explain which application each observation concerns.

The root `.archbase/scopes.yaml` is the only runtime scope index. It contains
`schemaVersion: 1` and a `scopes` list. Each entry has `scope.path`, `pattern.id`,
`pattern.source` and, for local bundles, `pattern.root`; `behavior` specifies local
customization and nearest-scope selection. Local roots must remain inside
`.archbase`. Stored bundles include their manifest and typed Markdown examples.
Legacy per-directory indices are not supported and must not be silently migrated.

For a target path, select the deepest matching directory scope in that root index.
Validate that selected entry and its local bundle. An invalid child cannot be hidden
by a valid ancestor. When uncertain about validity without `arc`, report the exact
unverified configuration and withhold affected conclusions rather than guessing.

Canonical rule documents are stored in `.archbase/rules`. Agent entry files contain
managed indices and may also contain independent framework or user instructions.
Read these instructions as applicable project policy. Preserve external content.
Local rules and customized pattern examples override registry defaults. A catalog
rule that the project has not adopted is a candidate recommendation, not a finding.

The bundled catalog is a versioned reference. Its supported names do not guarantee
compatibility with every installed framework version. Explain observed differences.
In particular, `architecture/next-feature-app-router@1` describes App Router feature
boundaries, while `architecture/next-modular@1` describes the page/hook/util model.
Do not mix their layouts without examining the actual router and local conventions.
