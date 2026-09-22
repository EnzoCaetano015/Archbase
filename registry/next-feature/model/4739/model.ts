export type Item = {
  id: string
  name: string
  ownerId: string
}

export function normalizeItemName(value: string): string {
  return value.trim().replace(/\s+/g, " ")
}
