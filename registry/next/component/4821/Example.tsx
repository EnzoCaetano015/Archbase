export type ExampleProps = {
  label: string
}

export function Example({ label }: ExampleProps) {
  return <span>{label}</span>
}
