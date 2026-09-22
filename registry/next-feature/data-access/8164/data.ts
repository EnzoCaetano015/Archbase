import "server-only"
import { database } from "@/shared/db/database"
import type { Item } from "../model/items"

export async function findItemById(id: string): Promise<Item | null> {
  return database.item.findUnique({
    where: { id },
    select: { id: true, name: true, ownerId: true },
  })
}

export async function insertItem(input: Omit<Item, "id">): Promise<Item> {
  return database.item.create({
    data: input,
    select: { id: true, name: true, ownerId: true },
  })
}
