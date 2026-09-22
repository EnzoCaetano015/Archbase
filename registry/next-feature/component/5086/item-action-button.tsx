"use client"

import { useTransition } from "react"
import { archiveItemAction } from "../actions/items"

type ItemActionButtonProps = {
  itemId: string
}

export function ItemActionButton({ itemId }: ItemActionButtonProps) {
  const [isPending, startTransition] = useTransition()

  return (
    <button
      type="button"
      disabled={isPending}
      onClick={() => startTransition(() => archiveItemAction({ itemId }))}
    >
      {isPending ? "Archiving..." : "Archive"}
    </button>
  )
}
