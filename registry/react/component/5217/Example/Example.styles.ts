import type { SxProps, Theme } from "@mui/material"

export const root: SxProps<Theme> = (theme) => ({
    display: "grid",
    gap: theme.spacing(1),
    padding: theme.spacing(2),
    border: `1px solid ${theme.palette.divider}`,
    borderRadius: theme.shape.borderRadius,
    backgroundColor: theme.palette.background.paper,
})

export const action: SxProps<Theme> = (theme) => ({
    marginTop: theme.spacing(1),
})
