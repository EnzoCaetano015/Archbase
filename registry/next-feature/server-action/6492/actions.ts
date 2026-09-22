"use server"

import { revalidatePath } from "next/cache"
import { requireUser } from "@/shared/auth/session"
import { createItem } from "../application/create-item"
import { createItemSchema } from "../schemas/items"

type CreateItemResult =
  | { ok: true; value: { id: string } }
  | { ok: false; error: string }

export async function createItemAction(input: unknown): Promise<CreateItemResult> {
  const parsed = createItemSchema.safeParse(input)
  if (!parsed.success) return { ok: false, error: "Invalid item" }

  const actor = await requireUser()
  const item = await createItem({ actorId: actor.id, ...parsed.data })

  revalidatePath("/items")
  return { ok: true, value: { id: item.id } }
}
