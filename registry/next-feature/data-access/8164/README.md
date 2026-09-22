# next-feature/data-access@8164

Persistence adapter owned by one feature.

- Keep the module server-only and isolate the database client here.
- Select fields explicitly and return model-shaped values.
- Do not import actions, route handlers, or React components.
