import { act, render, screen } from '@testing-library/react'
import { request } from '../../api/client'
import { AuthProvider } from './AuthProvider'
import { useSession } from './session'

const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })

function Probe() {
  const { user } = useSession()
  return <p>{user === undefined ? '確認中' : user === null ? '未ログイン' : user.id}</p>
}

describe('AuthProvider', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('GET /me が成功したらログイン中になる', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => json({ id: 'u1' })),
    )
    render(
      <AuthProvider>
        <Probe />
      </AuthProvider>,
    )
    expect(screen.getByText('確認中')).toBeInTheDocument()
    expect(await screen.findByText('u1')).toBeInTheDocument()
  })

  it('GET /me が 401 なら未ログインになる', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => json({ code: 'unauthorized', message: 'ログインが必要です' }, 401)),
    )
    render(
      <AuthProvider>
        <Probe />
      </AuthProvider>,
    )
    expect(await screen.findByText('未ログイン')).toBeInTheDocument()
  })

  it('ほかの API で 401 unauthorized が返ったら未ログインに戻す', async () => {
    const fetchMock = vi.fn(async () => json({ id: 'u1' }))
    vi.stubGlobal('fetch', fetchMock)
    render(
      <AuthProvider>
        <Probe />
      </AuthProvider>,
    )
    await screen.findByText('u1')
    fetchMock.mockResolvedValueOnce(
      json({ code: 'unauthorized', message: 'ログインが必要です' }, 401),
    )
    await act(() => request('/deals').catch(() => {}))
    expect(screen.getByText('未ログイン')).toBeInTheDocument()
  })
})
