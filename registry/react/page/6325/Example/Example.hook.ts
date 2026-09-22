import { zodResolver } from "@hookform/resolvers/zod"
import { useForm } from "react-hook-form"

import { useCreateExample } from "../../api/controllers/example"
import { exampleSchema, type ExampleFormValues } from "./Example.schema"

export const useExamplePage = () => {
    const createExample = useCreateExample()
    const { control, handleSubmit, reset } = useForm<ExampleFormValues>({
        defaultValues: { name: "" },
        resolver: zodResolver(exampleSchema),
    })

    const onSubmit = handleSubmit(async (values) => {
        await createExample.mutateAsync(values)
        reset()
    })

    return {
        control,
        isPending: createExample.isPending,
        onSubmit,
    }
}
