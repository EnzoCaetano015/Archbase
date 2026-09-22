import type { Metadata } from "next"
import { notFound } from "next/navigation"
import { findItem } from "@/features/catalog/application/queries"
import { ItemDetails } from "@/features/catalog/components/item-details"

type ItemPageProps = {
  params: Promise<{ id: string }>
}

export async function generateMetadata({ params }: ItemPageProps): Promise<Metadata> {
  const { id } = await params
  const item = await findItem(id)

  return item ? { title: item.name } : { title: "Item not found" }
}

export default async function ItemPage({ params }: ItemPageProps) {
  const { id } = await params
  const item = await findItem(id)
  if (!item) notFound()

  return <ItemDetails item={item} />
}
