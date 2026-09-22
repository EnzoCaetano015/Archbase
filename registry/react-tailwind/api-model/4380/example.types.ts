export enum ExampleQueryKeys {
    List = "examples_list",
}

export type Example = {
    id: number
    name: string
}

export type ListExamplesResponse = {
    items: Example[]
}

export type CreateExampleRequest = {
    name: string
}

export type CreateExampleResponse = {
    message: string
    example: Example
}
