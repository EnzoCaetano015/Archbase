# Modular Next architecture

This rule maps the initial Next structural patterns onto common project paths without copying their source examples.

Pages compose the user-facing feature and delegate their local state and effects to a page hook. Shared components remain presentation-focused and cannot depend on pages. Reusable hooks expose stateful behavior without rendering UI, while utilities contain deterministic transformations without state or side effects.

When creating or modifying matching code, resolve the applicable local scope first and inspect the associated pattern for its required file structure, allowed changes, and preserved responsibilities.
