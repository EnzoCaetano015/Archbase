import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { API_ROUTES } from "@/api/routes"
import type {
    CreateExampleRequest,
    CreateExampleResponse,
    ListExamplesResponse,
} from "@/api/models/example.types"
import { ExampleQueryKeys } from "@/api/models/example.types"
import { api } from "@/lib/config/axios"

export const useExamples = () => {
    return useQuery({
        queryKey: [ExampleQueryKeys.List],
        queryFn: async () => {
            const { data } = await api.get<ListExamplesResponse>(
                API_ROUTES.examples.list,
            )
            return data.items
        },
    })
}

export const useCreateExample = () => {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: async (request: CreateExampleRequest) => {
            const { data } = await api.post<CreateExampleResponse>(
                API_ROUTES.examples.create,
                request,
            )
            return data
        },
        onSuccess: async () => {
            await queryClient.invalidateQueries({
                queryKey: [ExampleQueryKeys.List],
            })
        },
    })
}
