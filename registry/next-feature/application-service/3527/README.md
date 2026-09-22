# next-feature/application-service@3527

Server-only application use case inside a feature module.

- Coordinate the workflow and domain policy here.
- Depend inward on `model` and outward through the feature's `data` boundary.
- Remain independent of route files, HTTP responses, React components, and cache revalidation.
