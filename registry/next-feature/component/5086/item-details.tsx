import type { Item } from "../model/items"
import { ItemActionButton } from "./item-action-button"

type ItemDetailsProps = {
  item: Item
}

export function ItemDetails({ item }: ItemDetailsProps) {
  return (
    <article>
      <h1>{item.name}</h1>
      <ItemActionButton itemId={item.id} />
    </article>
  )
}
