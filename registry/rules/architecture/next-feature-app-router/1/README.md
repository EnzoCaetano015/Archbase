# Feature-oriented Next.js App Router architecture

This variant is derived from the structural boundaries observed in the Devely reference project. It keeps framework routes under `app/` and organizes business capabilities under `features/<domain>/`.

The dependency flow is intentionally one-way: route and UI boundaries delegate to application services; application services coordinate domain policy and data access; data modules isolate persistence; model modules stay pure. Runtime schemas validate untrusted input before application workflows begin.

Server Components remain the default. Client Components are narrow interaction islands, UI mutations cross Server Actions, and Route Handlers are used only when an HTTP endpoint is genuinely required for an external consumer.

When creating or modifying matching code, resolve the nearest local scope first and inspect the associated pattern for its required structure, allowed changes, and preserved responsibilities.
