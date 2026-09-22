import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

import { useExamplePage } from "./Example.hook"

export const ExamplePage = () => {
    const { errors, isPending, onSubmit, register } = useExamplePage()

    return (
        <main className="mx-auto flex min-h-screen w-full max-w-2xl items-center px-4 py-10">
            <section className="w-full space-y-6 rounded-2xl border border-border bg-card p-6 text-card-foreground shadow-sm">
                <header className="space-y-2">
                    <h1 className="text-3xl font-bold tracking-tight">Create example</h1>
                    <p className="text-muted-foreground">
                        Complete the form to create a new example.
                    </p>
                </header>
                <form className="space-y-4" onSubmit={onSubmit}>
                    <div className="space-y-2">
                        <label htmlFor="example-name" className="text-sm font-medium">
                            Name
                        </label>
                        <Input
                            id="example-name"
                            disabled={isPending}
                            aria-invalid={Boolean(errors.name)}
                            aria-describedby={errors.name ? "example-name-error" : undefined}
                            {...register("name")}
                        />
                        {errors.name ? (
                            <p id="example-name-error" className="text-sm text-destructive">
                                {errors.name.message}
                            </p>
                        ) : null}
                    </div>
                    <Button type="submit" loading={isPending}>
                        Save
                    </Button>
                </form>
            </section>
        </main>
    )
}
