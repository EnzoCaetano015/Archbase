# python/repository@8462

Synchronous repository that isolates PostgreSQL access behind an injected database boundary.

- Change identifiers, imports, types, queries, row mapping, and the choice between read sessions and write transactions.
- Preserve parameterized queries, database injection, and managed session or transaction scopes.
- Keep business rules and external HTTP integrations outside the repository.
