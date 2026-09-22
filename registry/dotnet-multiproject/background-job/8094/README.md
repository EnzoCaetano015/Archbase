# dotnet-multiproject/background-job@8094

Scheduled job boundary suitable for Hangfire discovery and registration.

- Express scheduling as metadata separate from execution.
- Delegate use cases to application service contracts and propagate cancellation.
- Use structured logging and avoid direct controller or database dependencies.
