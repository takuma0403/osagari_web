import { render } from '@testing-library/react'
import type { ReactElement } from 'react'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { Destination } from './Destination'

type Options = {
  /** 画面に渡す location.state */
  state?: unknown
  /** 遷移先として用意するパス（見出しにパスを出す仮の画面になる） */
  destinations?: string[]
}

/**
 * 画面を path のルートとして描画する。画面からの遷移は destinations の仮の画面で確かめる。
 * 遷移したかは router.state.location、渡した state は見出しの data-state で見る。
 */
export function renderRoute(
  path: string,
  element: ReactElement,
  { state, destinations = [] }: Options = {},
) {
  const router = createMemoryRouter(
    [{ path, element }, ...destinations.map((to) => ({ path: to, element: <Destination /> }))],
    { initialEntries: [{ pathname: path, state }] },
  )
  return { router, ...render(<RouterProvider router={router} />) }
}
