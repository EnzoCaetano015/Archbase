import { useExample } from './Example.hook'

export function Example() {
  const view = useExample()

  return <main>{view.content}</main>
}
