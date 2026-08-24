import { useState } from 'react'

export type ExampleResult = {
  value: string
  setValue: (value: string) => void
}

export function useExample(): ExampleResult {
  const [value, setValue] = useState('')

  return { value, setValue }
}
