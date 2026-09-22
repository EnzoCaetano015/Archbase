# nestjs/guard@3642

Authentication boundary implementing NestJS `CanActivate`.

- Honor explicit public-route metadata before requiring credentials.
- Delegate token verification to the authentication service.
- Attach verified claims to the request and fail closed when credentials are absent or invalid.
