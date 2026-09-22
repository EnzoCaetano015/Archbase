# Modular React and Tailwind architecture

This variant models a React and Vite application built around Tailwind semantic tokens, shadcn-style primitives, typed TanStack Query controllers, centralized endpoint paths, and a page registry separated from the route tree.

Pages render and compose primitives while colocated hooks orchestrate forms and remote state. API controllers consume both the endpoint catalog and transport models. Domain components live above the reusable `components/ui` primitive layer. Shared hooks and utilities reuse the generic React patterns where their responsibilities do not depend on the styling system.

This rule intentionally coexists with `architecture/react-vite-modular@1`; projects choose the variant that matches their actual codebase. Agents should resolve the applicable scope and inspect each referenced pattern before producing code.
