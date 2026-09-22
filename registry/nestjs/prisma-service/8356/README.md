# nestjs/prisma-service@8356

Lifecycle-aware Prisma infrastructure provider for NestJS.

- Create one injectable client and expose it through an infrastructure module.
- Obtain connection configuration from `ConfigService`.
- Keep feature-specific queries in feature services or repositories, not in the shared client.
