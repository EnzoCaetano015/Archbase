export type ExampleInput = {
  value: string
}

export type ExampleOutput = {
  normalizedValue: string
}

export function createExample(input: ExampleInput): ExampleOutput {
  return { normalizedValue: input.value.trim() }
}
