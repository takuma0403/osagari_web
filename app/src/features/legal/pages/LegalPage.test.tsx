import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { routes } from '../../../routes'
import { SessionContext, type Session } from '../../auth/session'

const me: Session['user'] = {
  id: 'u1',
  email: '26238001@edu.cc.saga-u.ac.jp',
  studentNumber: '26238001',
  admissionYear: 2026,
  departmentCode: '238',
  department: '238',
  departmentName: '理工学部 理工学科',
  expiresOn: '2030-03-31',
  status: 'active',
  sellerStatus: 'approved',
}

// ルート（routes.tsx）ごと描画し、本物の本文（docs/legal の正本）が載ることを確かめる
// 本文は use() で読み込みを待つ（Suspense）。中断から再開させるため、描画は非同期の act で包む
async function setup(path: string, user: Session['user'] = null) {
  const router = createMemoryRouter(routes, { initialEntries: [path] })
  const session: Session = { user, signIn: vi.fn(), signOut: vi.fn() }
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

  it('戻るで開いた元の画面（新規登録）へ戻る', async () => {
    const { router, user } = await setup('/signup')
    await user.click(screen.getByRole('link', { name: '利用規約' }))
    expect(router.state.location.pathname).toBe('/terms')
    await user.click(screen.getByRole('button', { name: '戻る' }))
    expect(router.state.location.pathname).toBe('/signup')
  })

  it('URL を直接開いたときの戻るは商品一覧へ', async () => {
    // 商品一覧が読み込みで呼ぶ API は応答を返さずに待たせる（ここでは行き先だけを確かめる）
    vi.stubGlobal(
      'fetch',
      vi.fn(() => new Promise<Response>(() => {})),
    )
    const { router, user } = await setup('/terms', me)
    await user.click(screen.getByRole('button', { name: '戻る' }))
    expect(router.state.location.pathname).toBe('/')
    vi.unstubAllGlobals()
  })

  it('未ログインで URL を直接開いたときの戻るはログイン画面へ', async () => {
    const { router, user } = await setup('/terms')
    await user.click(screen.getByRole('button', { name: '戻る' }))
    expect(router.state.location.pathname).toBe('/login')
  })
})
