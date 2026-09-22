import { Button, Stack, TextField, Typography } from "@mui/material"
import { Controller } from "react-hook-form"

import * as styles from "./Example.styles"
import { useExamplePage } from "./Example.hook"

export const ExamplePage = () => {
    const { control, isPending, onSubmit } = useExamplePage()

    return (
        <Stack component="main" sx={styles.root}>
            <Typography component="h1" variant="h4">
                Create example
            </Typography>
            <Stack component="form" onSubmit={onSubmit} sx={styles.form}>
                <Controller
                    name="name"
                    control={control}
                    render={({ field, fieldState }) => (
                        <TextField
                            {...field}
                            label="Name"
                            disabled={isPending}
                            error={Boolean(fieldState.error)}
                            helperText={fieldState.error?.message}
                        />
                    )}
                />
                <Button type="submit" variant="contained" disabled={isPending}>
                    {isPending ? "Saving..." : "Save"}
                </Button>
            </Stack>
        </Stack>
    )
}
