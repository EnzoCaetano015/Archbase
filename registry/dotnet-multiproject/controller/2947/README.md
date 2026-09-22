# dotnet-multiproject/controller@2947

Thin ASP.NET Core controller for a contract-driven application layer.

- Depend on an application service interface, not a concrete service or repository.
- Accept transport contracts and return explicit HTTP results.
- Propagate cancellation and keep persistence details outside the API project.
