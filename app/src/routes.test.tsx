import { render, screen } from '@testing-library/react'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { routes } from './routes'

function renderAt(path: string) {
  const router = createMemoryRouter(routes, { initialEntries: [path] })
  render(<RouterProvider router={router} />)
  return router
}

describe('routes', () => {
  it.each([
    ['/login', 'ログイン'],
    ['/signup', '新規登録'],
    ['/signup/code', '認証コード入力（新規登録）'],
    ['/signup/password', 'パスワード設定'],
    ['/password-reset', 'パスワード再設定'],
    ['/password-reset/code', '認証コード入力（パスワード再設定）'],
    ['/password-reset/password', '新しいパスワードの設定'],
    ['/recheck', 'ログインの確認'],
    ['/', '商品一覧'],
    ['/search', '検索結果'],
    ['/items/abc123', '商品詳細'],
    ['/items/abc123/request', '購入希望・決済'],
    ['/items/abc123/edit', '出品の編集'],
    ['/sell', '出品'],
    ['/deals', '取引一覧'],
    ['/deals/42', '取引詳細'],
    ['/deals/42/scan', 'QR読み取り'],
    ['/deals/42/complete', '受け渡しの完了'],
    ['/mypage', 'マイページ'],
    ['/mypage/student-card', '学生証登録'],
    ['/mypage/sales', '売上管理'],
    ['/admin/student-cards', '学生証審査'],
    ['/admin/users', 'ユーザー'],
    ['/admin/items', '出品・取引'],
    ['/admin/settings', '設定'],
  ])('%s で「%s」を表示する', (path, title) => {
    renderAt(path)
    expect(screen.getByRole('heading', { level: 1, name: title })).toBeInTheDocument()
  })

  it('管理画面のルートでは管理メニューを表示する', () => {
    renderAt('/admin/users')
    expect(screen.getByRole('navigation', { name: '管理メニュー' })).toBeInTheDocument()
  })

  it('/admin は学生証審査に移る', () => {
    const router = renderAt('/admin')
    expect(router.state.location.pathname).toBe('/admin/student-cards')
    expect(screen.getByRole('heading', { level: 1, name: '学生証審査' })).toBeInTheDocument()
  })

  it('知らないパスは「ページが見つかりません」を表示する', () => {
    renderAt('/no-such-page')
    expect(
      screen.getByRole('heading', { level: 1, name: 'ページが見つかりません' }),
    ).toBeInTheDocument()
    expect(screen.getByRole('link', { name: '商品一覧へ戻る' })).toHaveAttribute('href', '/')
  })
})
