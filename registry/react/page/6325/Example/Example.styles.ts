import type { SxProps, Theme } from "@mui/material"

export const root: SxProps<Theme> = (theme) => ({
    width: "100%",
    maxWidth: 640,
    gap: theme.spacing(3),
    marginInline: "auto",
    padding: theme.spacing(3),
})

export const form: SxProps<Theme> = (theme) => ({
    gap: theme.spacing(2),
})
