# python/service@7315

Application service that owns orchestration and transformations while depending on a repository through constructor injection.

- Change identifiers, imports, types, application logic, transformations, and injected dependencies.
- Preserve constructor injection, repository delegation, and the explicit FastAPI dependency factory.
- Keep SQL and HTTP transport decisions outside the service.
