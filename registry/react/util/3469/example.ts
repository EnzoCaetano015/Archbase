export type FormatExampleOptions = {
    fallback?: string
}

export const formatExample = (
    value: string | null | undefined,
    options: FormatExampleOptions = {},
): string => {
    const normalized = value?.trim()
    return normalized || options.fallback || "—"
}
