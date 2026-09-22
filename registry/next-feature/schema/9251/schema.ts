import { z } from "zod"

export const createItemSchema = z.object({
  name: z.string().trim().min(1, "Name is required").max(100),
})

export type CreateItemInput = z.infer<typeof createItemSchema>
