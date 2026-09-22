import { defineCollection, z } from "astro:content"

const examples = defineCollection({
	type: "data",
	schema: z.object({
		title: z.string(),
		description: z.string(),
		image: z.string(),
	}),
})

export const collections = {
	examples,
}
