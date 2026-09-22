import { Button } from "@/components/ui/button"

import type { ExampleProps } from "./Example.types"

export const Example = ({
    title,
    description,
    loading = false,
    onAction,
}: ExampleProps) => {
    return (
        <article className="space-y-4 rounded-xl border border-border bg-card p-6 text-card-foreground">
            <div className="space-y-1">
                <h2 className="text-xl font-semibold">{title}</h2>
                {description ? (
                    <p className="text-sm text-muted-foreground">{description}</p>
                ) : null}
            </div>
            <Button type="button" loading={loading} onClick={onAction}>
                Continue
            </Button>
        </article>
    )
}
