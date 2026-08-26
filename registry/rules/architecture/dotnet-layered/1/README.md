# Layered .NET architecture

This rule connects the initial controller, service, and repository patterns without reproducing their C# examples.

Controllers own HTTP translation and delegate work. Services coordinate application behavior through injected abstractions. Repositories isolate persistence concerns. Dependencies move in one direction from Controller to Service to Repository; lower layers do not import or call higher layers.

Agents should resolve the applicable scope and inspect the referenced pattern before producing code for any layer.
