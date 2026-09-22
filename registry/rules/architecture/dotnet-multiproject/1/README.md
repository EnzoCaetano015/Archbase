# Multi-project ASP.NET Core API architecture

This variant is derived from the structure observed in the Todo API reference: separate API, service, data, model, job, and utility projects; interface-driven services and repositories; PetaPoco with SQLKata; centralized dependency injection and exception handling; and Hangfire-compatible jobs.

The registry variant normalizes those responsibilities into `*.Api`, `*.Application`, `*.Infrastructure`, `*.Contracts`, and `*.Jobs` projects under `src`. Dependencies flow inward through interfaces and shared contracts, while the API remains the composition root.

When creating or modifying matching code, resolve the nearest local scope first and inspect the associated pattern for its required structure, allowed changes, and preserved responsibilities.
