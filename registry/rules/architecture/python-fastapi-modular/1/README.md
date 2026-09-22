# Modular Python FastAPI architecture

This rule maps a feature-oriented FastAPI module to separate transport, contract, application, and persistence patterns without reproducing their Python examples.

Routers translate HTTP input and output, schemas express Pydantic contracts, services coordinate application behavior, and repositories isolate parameterized PostgreSQL access. Dependencies move in one direction from Router to Service to Repository; schemas remain independent contracts and lower layers never import higher ones.

A module may omit a layer that has no responsibility. Agents should resolve the applicable scope and inspect the referenced pattern before creating or modifying a layer.
