import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { RequireAdmin } from './RequireAdmin'

const json = (body: unknown, status: number) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })

function renderGuard() {
  const router = createMemoryRouter(
    [
      {
        element: <RequireAdmin />,
        children: [{ path: '/admin', element: <h1>管理画面の中身</h1> }],
      },
    ],
    { initialEntries: ['/admin'] },
  )
  return render(<RouterProvider router={router} />)
}

describe('RequireAdmin', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('確認中は何も出さず、管理者なら中身を出す', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => new Response(null, { status: 204 })),
    )
    const { container } = renderGuard()
    expect(container).toBeEmptyDOMElement()
    expect(await screen.findByRole('heading', { name: '管理画面の中身' })).toBeInTheDocument()
  })

  it('管理者でなければ（404）「ページが見つかりません」を出し、中身は出さない', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => json({ code: 'not_found', message: '見つかりません' }, 404)),
    )
    renderGuard()
    expect(
      await screen.findByRole('heading', { level: 1, name: 'ページが見つかりません' }),
    ).toBeInTheDocument()
    expect(screen.getByRole('link', { name: '商品一覧へ戻る' })).toHaveAttribute('href', '/')
    expect(screen.queryByRole('heading', { name: '管理画面の中身' })).not.toBeInTheDocument()
  })

  it('通信できなければエラーを出し、読み込み直せる', async () => {
    const fetchMock = vi
      .fn()
      .mockRejectedValueOnce(new TypeError('Failed to fetch'))
      .mockResolvedValueOnce(new Response(null, { status: 204 }))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    renderGuard()
    expect(await screen.findByRole('alert')).toHaveTextContent('通信できませんでした')
    await user.click(screen.getByRole('button', { name: 'もう一度読み込む' }))
    expect(await screen.findByRole('heading', { name: '管理画面の中身' })).toBeInTheDocument()
  })
})
