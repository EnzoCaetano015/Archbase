import { createItem } from "@/features/catalog/application/create-item"
import { createItemSchema } from "@/features/catalog/schemas/items"

export async function POST(request: Request) {
  const parsed = createItemSchema.safeParse(await request.json())
  if (!parsed.success) {
    return Response.json({ error: "Invalid request" }, { status: 400 })
  }

  const item = await createItem(parsed.data)
  return Response.json(item, { status: 201 })
}
