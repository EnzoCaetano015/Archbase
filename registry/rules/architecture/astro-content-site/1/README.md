# Astro content site architecture

This rule models a content-oriented Astro site with file-based routes, shared document layouts, server-rendered `.astro` components, validated content collections, Tailwind presentation, and pure typed utilities.

Astro ships components without client JavaScript by default. Browser scripts and framework hydration are explicit progressive-enhancement boundaries rather than the default rendering model. Layouts own the shared document shell, pages own route composition and content queries, and collections keep validated data separate from presentation.

Agents should resolve the applicable scope and inspect each referenced pattern before creating or modifying code.
