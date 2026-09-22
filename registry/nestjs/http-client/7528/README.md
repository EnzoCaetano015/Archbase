# nestjs/http-client@7528

Dedicated NestJS adapter for an external HTTP API.

- Configure base URLs and credentials through `ConfigService`.
- Encapsulate remote paths and response contracts behind an injectable client.
- Export the adapter rather than the raw `HttpService` to consuming features.
