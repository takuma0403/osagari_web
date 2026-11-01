import { render, screen } from '@testing-library/react'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { SessionContext, type Session } from './features/auth/session'
import { acknowledgeTrialNotice } from './lib/trialNotice'
import { routes } from './routes'

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

/** user を省くとログイン中。undefined を渡すと確認中 */
function renderAt(path: string, options: { user?: Session['user']; state?: unknown } = {}) {
  const { state } = options
  const user = 'user' in options ? options.user : me
  const router = createMemoryRouter(routes, { initialEntries: [{ pathname: path, state }] })
  render(
    <SessionContext value={{ user, signIn: vi.fn(), signOut: vi.fn() }}>
      <RouterProvider router={router} />
    </SessionContext>,
  )
  return router
}

describe('routes', () => {
  // 管理者の確認（GET /api/admin/session）は管理者として 204 を返す。
  // ほかに画面が読み込みで呼ぶ API は応答を返さずに待たせる（ここではルートと見出しだけを確かめる）
  // 試験運用の初回ダイアログは確認済みにしておく（出ることは最後の検査で確かめる）
  beforeEach(() => {
    acknowledgeTrialNotice()
    vi.stubGlobal(
      'fetch',
      vi.fn((url: string) =>
        url === '/api/admin/session'
          ? Promise.resolve(new Response(null, { status: 204 }))
          : new Promise<Response>(() => {}),
      ),
    )
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    localStorage.clear()
  })

  it.each([
    ['/', '新着の出品'],
    ['/search', '検索結果'],
    ['/items/abc123/request', '購入希望を出す'],
    ['/sell', '出品する'],
    ['/sell/i1', '出品を編集'],
    ['/sell/i1/requests', '取引詳細'],
    ['/deals', '取引'],
    ['/deals/42', '取引詳細'],
    ['/deals/42/scan', 'QRを読み取る'],
    ['/deals/42/complete', '受け渡しの完了'],
    ['/mypage', 'マイページ'],
    ['/mypage/student-card', '学生証の登録'],
    ['/mypage/sales', '売上管理'],
    ['/admin/student-cards', '学生証審査'],
    ['/admin/users', 'ユーザー'],
    ['/admin/items', '出品・取引'],
    ['/admin/settings', '設定'],
    ['/terms', '利用規約'],
    ['/privacy', 'プライバシーポリシー'],
  ])('ログイン中は %s で「%s」を表示する', async (path, title) => {
    renderAt(path)
    expect(await screen.findByRole('heading', { level: 1, name: title })).toBeInTheDocument()
  })

  it.each([
    ['/login', '佐賀大生どうしで、お下がりを回そう。'],
    ['/signup', '新規登録'],
    ['/password-reset', 'パスワードの再設定'],
    ['/signup/password', '新規登録'],
    ['/password-reset/password', 'パスワードの再設定'],
    ['/terms', '利用規約'],
    ['/privacy', 'プライバシーポリシー'],
  ])('未ログインでも %s で「%s」を表示する', (path, title) => {
    renderAt(path, { user: null })
    expect(screen.getByRole('heading', { level: 1, name: title })).toBeInTheDocument()
  })

  it.each(['/', '/items/abc123', '/deals/42?tab=qr', '/admin/users'])(
    '未ログインで %s を開くとログイン画面へ送り、開こうとしたページを残す',
    (path) => {
      const router = renderAt(path, { user: null })
      expect(router.state.location.pathname).toBe('/login')
      expect(router.state.location.state).toEqual({ from: path })
    },
  )

  it('ログイン状態の確認中は保護ルートもログイン画面も描画しない', () => {
    const router = renderAt('/deals', { user: undefined })
    expect(router.state.location.pathname).toBe('/deals')
    expect(screen.queryByRole('heading')).not.toBeInTheDocument()
  })

  it.each(['/login', '/signup', '/signup/code', '/password-reset', '/recheck'])(
    'ログイン済みで %s を開くと商品一覧へ送る',
    (path) => {
      const router = renderAt(path)
      expect(router.state.location.pathname).toBe('/')
    },
  )

  it('ログイン済みになったら、開こうとしていたページへ送る', () => {
    const router = renderAt('/login', { state: { from: '/deals/42' } })
    expect(router.state.location.pathname).toBe('/deals/42')
  })

  it('開こうとしていたページが外部のURLなら商品一覧へ送る', () => {
    const router = renderAt('/login', { state: { from: '//evil.example.com' } })
    expect(router.state.location.pathname).toBe('/')
  })

  it.each(['/', '/search', '/deals', '/mypage'])('%s では下部ナビを表示する', (path) => {
    renderAt(path)
    expect(screen.getByRole('navigation', { name: 'メインメニュー' })).toBeInTheDocument()
  })

  it.each(['/items/abc123', '/sell', '/deals/42', '/mypage/sales'])(
    '%s では下部ナビを表示しない',
    (path) => {
      renderAt(path)
      expect(screen.queryByRole('navigation', { name: 'メインメニュー' })).not.toBeInTheDocument()
    },
  )

  it('ログイン画面では下部ナビを表示しない', () => {
    renderAt('/login', { user: null })
    expect(screen.queryByRole('navigation', { name: 'メインメニュー' })).not.toBeInTheDocument()
  })

  it('管理画面のルートでは管理メニューを表示する', async () => {
    renderAt('/admin/users')
    expect(await screen.findByRole('navigation', { name: '管理メニュー' })).toBeInTheDocument()
  })

  it('/admin は学生証審査に移る', async () => {
    const router = renderAt('/admin')
    expect(await screen.findByRole('heading', { level: 1, name: '学生証審査' })).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/admin/student-cards')
  })

  it('管理者でなければ管理画面のルートでも「ページが見つかりません」を表示する', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () =>
        Response.json({ code: 'not_found', message: '見つかりません' }, { status: 404 }),
      ),
    )
    renderAt('/admin/student-cards')
    expect(
      await screen.findByRole('heading', { level: 1, name: 'ページが見つかりません' }),
    ).toBeInTheDocument()
    expect(screen.queryByRole('navigation', { name: '管理メニュー' })).not.toBeInTheDocument()
  })

  it('知らないパスは「ページが見つかりません」を表示する', () => {
    renderAt('/no-such-page', { user: null })
    expect(
      screen.getByRole('heading', { level: 1, name: 'ページが見つかりません' }),
    ).toBeInTheDocument()
    expect(screen.getByRole('link', { name: '商品一覧へ戻る' })).toHaveAttribute('href', '/')
  })

  it.each([
    ['/login', null],
    ['/', me],
    ['/no-such-page', null],
  ])('未確認の端末では %s で試験運用の初回ダイアログを出す', (path, user) => {
    localStorage.clear()
    renderAt(path, { user })
    expect(screen.getByRole('dialog', { name: '試験運用中です' })).toBeInTheDocument()
  })

  it.each(['/terms', '/privacy'])('未確認の端末でも %s では初回ダイアログを出さない', (path) => {
    localStorage.clear()
    renderAt(path, { user: null })
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })
})
