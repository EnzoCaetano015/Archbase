# dotnet-multiproject/exception-handler@8463

Global exception-to-HTTP mapping for ASP.NET Core.

- Map known failures deliberately and return a safe generic message for unexpected exceptions.
- Log the original exception with structured context.
- Preserve cancellation and keep feature workflows outside the handler.
