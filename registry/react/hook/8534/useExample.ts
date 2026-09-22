import { useCallback, useState } from "react"

export type UseExampleResult = {
    isOpen: boolean
    open: () => void
    close: () => void
}

export const useExample = (): UseExampleResult => {
    const [isOpen, setIsOpen] = useState(false)

    const open = useCallback(() => setIsOpen(true), [])
    const close = useCallback(() => setIsOpen(false), [])

    return {
        isOpen,
        open,
        close,
    }
}
