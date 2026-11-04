# Archbase documentation portal

React/Vite portal adapted from the supplied Figma export. Public copy stays in
English. Canonical rule IDs, mappings and all available declared pattern files
are read from `../registry/` by `scripts/generate-catalog.mjs`; the generated JSON
must never be edited manually. Optional missing pattern files are omitted.

Requires Node.js 24 or newer.

```sh
cd site
npm ci
npm run build
npm run check
npm run preview
```

The preview binds to localhost. The static output is `dist/`; known documentation
and compatibility routes have their own HTML with readable content and metadata,
so direct URLs work on GitHub Pages. Search and navigation run locally in the
browser. No Figma runtime, analytics, API or backend is required.

`public/install.sh` and `public/install.ps1` are byte-for-byte copies of the
existing canonical `site/install.*` files. The content check fails if they drift.
The original repository logo remains in the home hero, header and footer.

Skill availability and prices were checked with the authenticated Agensi creator
connector on November 4, 2026: all 11 live/approved. URLs and content versions are
in `src/content.ts`; Agensi listing-version labels are separate.

No deployment is needed for local review. The prepared Pages workflow publishes
only after a future authorized push; no commit or publication is part of this
preview delivery.
