# next-feature/server-action@6492

Mutation boundary for interactions initiated by the Next.js UI.

- Keep `"use server"` at the module boundary.
- Accept `unknown`, validate it, and establish the actor before invoking the use case.
- Return a serializable result and revalidate only affected paths or tags.
- Never access the database directly from the action.
