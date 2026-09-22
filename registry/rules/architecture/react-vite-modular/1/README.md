# Modular React and Vite architecture

This rule maps a feature-oriented React application to independent patterns for pages, reusable UI, API controllers, transport models, shared hooks, utilities, and routes without copying their source examples.

Pages render and compose while colocated hooks orchestrate page behavior. API controllers own TanStack Query hooks and HTTP access; models own contracts and query keys. Shared components stay presentation-focused, shared hooks own reusable state, utilities remain pure, and the route tree stays centralized.

Optional colocated files should be introduced only when their responsibility improves clarity or reuse. Agents should resolve the applicable scope and inspect its pattern before creating or modifying code.
