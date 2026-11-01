import { render } from '@testing-library/react'
import type { ReactElement } from 'react'
import { createMemoryRouter, parsePath, RouterProvider } from 'react-router'
import { SessionContext, type Session } from '../features/auth/session'
import { Destination } from './Destination'

type Options = {
  /** 開く URL（?以降のクエリも書ける）。path にパラメーター（:itemId など）があるときに渡す。既定は path */
  at?: string
  /** 画面に渡す location.state */
  state?: unknown
  /** 遷移先として用意するパス（見出しにパスを出す仮の画面になる） */
  destinations?: string[]
  /** ログイン状態。既定は未ログインで、signIn・signOut は呼ばれたかを確かめられる vi.fn */
  session?: Partial<Session>
}

/**
 * 画面を path のルートとして描画する。画面からの遷移は destinations の仮の画面で確かめる。
 * 遷移したかは router.state.location、渡した state は見出しの data-state で見る。
 */
export function renderRoute(
  path: string,
  element: ReactElement,
  { at, state, destinations = [], session: overrides }: Options = {},
) {
  const router = createMemoryRouter(
    [{ path, element }, ...destinations.map((to) => ({ path: to, element: <Destination /> }))],
    { initialEntries: [{ ...parsePath(at ?? path), state }] },
  )
  const session: Session = { user: null, signIn: vi.fn(), signOut: vi.fn(), ...overrides }
  return {
    router,
    session,
    ...render(
      <SessionContext value={session}>
        <RouterProvider router={router} />
      </SessionContext>,
    ),
  }
}
