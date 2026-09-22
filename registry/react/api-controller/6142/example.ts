import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { api } from "../../lib/config/api"
import {
    exampleKeys,
    type CreateExampleRequest,
    type CreateExampleResponse,
    type ListExamplesResponse,
} from "../models/example.types"

export const useExamples = () => {
    return useQuery({
        queryKey: exampleKeys.all,
        queryFn: async () => {
            const { data } = await api.get<ListExamplesResponse>("/examples")
            return data.data ?? []
        },
    })
}

export const useCreateExample = () => {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: async (request: CreateExampleRequest) => {
            const { data } = await api.post<CreateExampleResponse>(
                "/examples",
                request,
            )
            return data
        },
        onSuccess: async () => {
            await queryClient.invalidateQueries({ queryKey: exampleKeys.all })
        },
    })
}
