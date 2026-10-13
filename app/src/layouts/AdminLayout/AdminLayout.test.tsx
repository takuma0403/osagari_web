import { render, screen } from '@testing-library/react'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { AdminLayout } from './AdminLayout'

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
})
