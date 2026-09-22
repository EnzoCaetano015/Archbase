# Modular NestJS and Prisma API architecture

This variant is derived from the structural boundaries observed in the HubSpot Tickets Manager API: NestJS modules, decorator-based controllers, injectable services, validated DTOs, a global authentication guard, a configured HTTP integration, Prisma infrastructure, and global response policies.

The registry variant normalizes domain code under `src/features`, external systems under `src/integrations`, Prisma under `src/infrastructure`, and shared HTTP policies under `src/common/http`. This keeps framework composition explicit while preventing controllers and cross-cutting concerns from absorbing business behavior.

When creating or modifying matching code, resolve the nearest local scope first and inspect the associated pattern for its required structure, allowed changes, and preserved responsibilities.
