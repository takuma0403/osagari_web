import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { AdminLayout } from './AdminLayout'
import { useAdminLayout } from './adminLayoutContext'

/** 子の画面から件数を設定する */
function BadgeSetter() {
  const { setMenuBadge } = useAdminLayout()
  return (
    <>
      <button type="button" onClick={() => setMenuBadge('studentCards', 12)}>
        12件にする
      </button>
      <button type="button" onClick={() => setMenuBadge('studentCards', 0)}>
        0件にする
      </button>
    </>
  )
}

describe('AdminLayout', () => {
  it('管理メニューを表示し、現在のページを選択中にする', () => {
    const router = createMemoryRouter(
      [
        {
          element: <AdminLayout />,
          children: [{ path: '/admin/users', element: <h1>ユーザー一覧</h1> }],
        },
      ],
      { initialEntries: ['/admin/users'] },
    )
    render(<RouterProvider router={router} />)

    const nav = screen.getByRole('navigation', { name: '管理メニュー' })
    expect(nav).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'ユーザー' })).toHaveAttribute('aria-current', 'page')
    expect(screen.getByRole('link', { name: '学生証審査' })).not.toHaveAttribute('aria-current')
    expect(screen.getByRole('main')).toHaveTextContent('ユーザー一覧')
  })

  it('子の画面が設定した件数をメニューのバッジに出し、0件なら消す', async () => {
    const router = createMemoryRouter(
      [
        {
          element: <AdminLayout />,
          children: [{ path: '/admin/student-cards', element: <BadgeSetter /> }],
        },
      ],
      { initialEntries: ['/admin/student-cards'] },
    )
    const user = userEvent.setup()
    render(<RouterProvider router={router} />)

    expect(screen.getByRole('link', { name: '学生証審査' })).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: '12件にする' }))
    expect(screen.getByRole('link', { name: '学生証審査 審査待ち12件' })).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: '0件にする' }))
    expect(screen.getByRole('link', { name: '学生証審査' })).toBeInTheDocument()
  })
})
