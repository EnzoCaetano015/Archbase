# next-feature/route-handler@7318

App Router HTTP endpoint intended for external integrations or public APIs.

- Parse and validate transport input at the boundary.
- Delegate workflows to the feature application layer.
- Map outcomes to explicit HTTP responses without embedding persistence logic.
- Prefer Server Components for internal reads and Server Actions for UI mutations.
