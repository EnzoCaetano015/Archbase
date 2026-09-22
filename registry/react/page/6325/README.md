# react/page@6325

React page bundle that separates rendering from orchestration, validation, and larger theme-aware styles.

- Change identifiers, imports, content, orchestration, validation, and styles.
- Preserve a thin page component, a colocated page hook, and API access through controllers.
- Derive render state directly instead of duplicating it through effects.
- Schema and style files are optional; keep only the layers the page actually needs.
