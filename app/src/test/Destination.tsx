import { useLocation } from 'react-router'

/** テストで遷移先として置く仮の画面。見出しにパスを出し、受け取った state を data-state に入れる */
export function Destination() {
  const location = useLocation()
  return <h1 data-state={JSON.stringify(location.state ?? null)}>{location.pathname}</h1>
}
