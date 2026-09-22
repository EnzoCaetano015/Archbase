# nestjs/http-envelope@5281

Global success and error response normalization for NestJS HTTP applications.

- Wrap successful values without changing feature services.
- Treat caught values as `unknown` and expose a safe fallback for unexpected failures.
- Preserve the HTTP status of known `HttpException` values and keep feature rules outside the filter.
