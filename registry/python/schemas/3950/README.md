# python/schemas@3950

Pydantic contracts that keep input validation separate from transport, application, and persistence behavior.

- Change identifiers, fields, types, validation constraints, and model configuration.
- Preserve distinct request and response contracts and structural validation only.
- Do not query persistence or import the router from schema modules.
