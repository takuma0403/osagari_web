import { render, screen } from '@testing-library/react'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { UserLayout } from './UserLayout'

function renderLayout(nav?: boolean) {
  const router = createMemoryRouter([
    {
      element: <UserLayout nav={nav} />,
      children: [{ path: '/', element: <p>本文</p> }],
    },
  ])
  render(<RouterProvider router={router} />)
}

/** 画面幅が PC（1024px 以上）かどうかを matchMedia で偽装する */
function mockPc(matches: boolean) {
  vi.stubGlobal(
    'matchMedia',
    vi.fn((query: string) => ({
      matches,
      media: query,
      addEventListener: () => {},
      removeEventListener: () => {},
    })),
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('UserLayout', () => {
  it('子ルートを main の中に表示する', () => {
    renderLayout()
    expect(screen.getByRole('main')).toHaveTextContent('本文')
  })

  it('nav を付けたときだけナビを出す', () => {
    renderLayout(true)
    expect(screen.getByRole('navigation', { name: 'メインメニュー' })).toBeInTheDocument()
  })

  it('nav がなければナビを出さない', () => {
    renderLayout()
    expect(screen.queryByRole('navigation')).not.toBeInTheDocument()
  })

  it('スマホの幅ではナビを本文の後ろ（下部）に置く', () => {
    mockPc(false)
    renderLayout(true)
    const nav = screen.getByRole('navigation', { name: 'メインメニュー' })
    expect(screen.getByRole('main').compareDocumentPosition(nav)).toBe(
      Node.DOCUMENT_POSITION_FOLLOWING,
    )
    expect(screen.queryByText('佐賀大お下がりフリマ')).not.toBeInTheDocument()
  })

  it('PC の幅ではナビを本文の前（上部）に置き、ロゴとアプリ名を出す', () => {
    mockPc(true)
    renderLayout(true)
    const nav = screen.getByRole('navigation', { name: 'メインメニュー' })
    expect(screen.getByRole('main').compareDocumentPosition(nav)).toBe(
      Node.DOCUMENT_POSITION_PRECEDING,
    )
    expect(nav).toHaveTextContent('佐賀大お下がりフリマ')
  })
})
