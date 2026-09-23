# spring-boot/security@8196

Stateless Spring Security and JWT authentication boundary.

- Declare public routes explicitly and require authentication everywhere else.
- Validate tokens before populating the security context.
- Continue the filter chain exactly once and keep authorization policy out of token parsing.
