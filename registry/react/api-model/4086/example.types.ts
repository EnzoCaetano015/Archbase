import type { ApiResponse } from "../../lib/types/api"

export type Example = {
    id: number
    name: string
}

export type CreateExampleRequest = {
    name: string
}

export type CreateExampleResponse = ApiResponse<Example>

export type ListExamplesResponse = ApiResponse<Example[]>

export const exampleKeys = {
    all: ["examples"] as const,
    detail: (id: number) => ["examples", id] as const,
}
