import { Box, Typography } from "@mui/material"

import * as styles from "./Example.styles"
import type { ExampleProps } from "./Example.types"

export const Example = ({ title, description, action }: ExampleProps) => {
    return (
        <Box component="section" sx={styles.root}>
            <Typography component="h2" variant="h6">
                {title}
            </Typography>
            {description ? (
                <Typography color="text.secondary">{description}</Typography>
            ) : null}
            {action ? <Box sx={styles.action}>{action}</Box> : null}
        </Box>
    )
}
