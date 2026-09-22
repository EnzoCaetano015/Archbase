export const API_ROUTES = {
    examples: {
        list: "/api/v1/examples",
        create: "/api/v1/examples",
        detail: (id: number) => `/api/v1/examples/${id}`,
    },
} as const
