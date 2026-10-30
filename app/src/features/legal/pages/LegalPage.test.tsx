import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { routes } from '../../../routes'
import { SessionContext, type Session } from '../../auth/session'

// ルート（routes.tsx）ごと描画し、本物の本文（docs/legal の正本）が載ることを確かめる
// 本文は use() で読み込みを待つ（Suspense）。中断から再開させるため、描画は非同期の act で包む
async function setup(path: string) {
  const router = createMemoryRouter(routes, { initialEntries: [path] })
  const session: Session = { user: null, signIn: vi.fn(), signOut: vi.fn() }
  await act(async () => {
    render(
      <SessionContext value={session}>
        <RouterProvider router={router} />
      </SessionContext>,
    )
  })
  return { router, user: userEvent.setup() }
}

describe('LegalPage', () => {
  it('/terms に利用規約の本文を出す（未ログインでも見られる）', async () => {
    await setup('/terms')
    expect(screen.getByRole('heading', { level: 1, name: '利用規約' })).toBeInTheDocument()
    expect(
      await screen.findByRole('heading', { name: '第１条（本サービスと運営者）' }),
    ).toBeInTheDocument()
    expect(screen.getByText(/佐賀大学の公式サービスではありません/)).toBeInTheDocument()
    expect(screen.queryByText(/正本です/)).not.toBeInTheDocument()
  })

  it('/privacy にプライバシーポリシーの本文と表を出す', async () => {
    await setup('/privacy')
    expect(
      screen.getByRole('heading', { level: 1, name: 'プライバシーポリシー' }),
    ).toBeInTheDocument()
    expect(
      await screen.findByRole('heading', { name: '第２条（取得する情報）' }),
    ).toBeInTheDocument()
    expect(screen.getAllByRole('table').length).toBeGreaterThan(0)
  })

  it('本文中のプライバシーポリシーへのリンクは SPA の中で /privacy を開く', async () => {
    const { router, user } = await setup('/terms')
    await user.click(await screen.findByRole('link', { name: 'プライバシーポリシー' }))
    expect(router.state.location.pathname).toBe('/privacy')
    expect(
      screen.getByRole('heading', { level: 1, name: 'プライバシーポリシー' }),
    ).toBeInTheDocument()
  })

  it.each([
    ['利用規約', '/terms'],
    ['プライバシーポリシー', '/privacy'],
  ])('ログイン画面の「%s」のリンクから %s を開ける', async (name, path) => {
    const { router, user } = await setup('/login')
    await user.click(screen.getByRole('link', { name }))
    expect(router.state.location.pathname).toBe(path)
    expect(screen.getByRole('heading', { level: 1, name })).toBeInTheDocument()
  })
})
