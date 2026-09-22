import "server-only"
import { insertItem } from "../data/items"
import { normalizeItemName, type Item } from "../model/items"

export type CreateItemInput = {
  actorId: string
  name: string
}

export async function createItem(input: CreateItemInput): Promise<Item> {
  const name = normalizeItemName(input.name)
  if (!name) throw new Error("Item name is required")

  return insertItem({ ownerId: input.actorId, name })
}
