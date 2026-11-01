import { act, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderRoute } from '../../../test/renderRoute'
import { RecheckPage } from './RecheckPage'

const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })

const me = { id: 'u1', email: '26238001@edu.cc.saga-u.ac.jp' }

const challenge = (reason: 'code_expired' | 'new_device' = 'new_device') => ({
  challengeToken: 'challenge-token',
  reason,
  email: '26238001@edu.cc.saga-u.ac.jp',
})

function setup(handler: (path: string) => Response, state: unknown = undefined) {
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => handler(String(input)))
  vi.stubGlobal('fetch', fetchMock)
  const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })
  const view = renderRoute('/recheck', <RecheckPage />, {
    state: state === undefined ? { challenge: challenge(), sentAt: Date.now() } : state,
    destinations: ['/login'],
  })
  return { ...view, user, fetchMock }
}

async function enterCode(user: ReturnType<typeof userEvent.setup>, code: string) {
  await user.click(screen.getByLabelText('1桁目'))
  await user.paste(code)
}

describe('RecheckPage', () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    vi.setSystemTime(new Date('2026-10-17T12:00:00+09:00'))
  })
  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('チャレンジがなければ /login へ戻す', () => {
    const { router } = setup(() => json({}), null)
    expect(router.state.location.pathname).toBe('/login')
  })

  it('はじめての端末の説明・送信先・残り時間を出す', () => {
    setup(() => json({}))
    expect(
      screen.getByRole('heading', { level: 1, name: '本人確認をお願いします' }),
    ).toBeInTheDocument()
    expect(screen.getByText(/はじめて使う端末からのログインのため/)).toBeInTheDocument()
    expect(screen.getByText('26238001@edu.cc.saga-u.ac.jp')).toBeInTheDocument()
    expect(screen.getByRole('group', { name: 'メールに届いた6桁のコード' })).toBeInTheDocument()
    expect(screen.getByText('10:00')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'コードを再送（1:00 後）' })).toBeDisabled()
    expect(screen.getByRole('link', { name: '別のアカウント' })).toHaveAttribute('href', '/login')
    expect(screen.getByText(/卒業などで大学のメールが受け取れなくなると/)).toBeInTheDocument()
  })

  it('30日たったときの説明を出す', () => {
    setup(() => json({}), { challenge: challenge('code_expired'), sentAt: Date.now() })
    expect(screen.getByText(/前回メールで確認してから30日たったため/)).toBeInTheDocument()
  })

  it('確認できたらログイン状態にする（パスワードは送らない）', async () => {
    const { user, session, fetchMock } = setup(() => json(me))
    await enterCode(user, '123456')
    await user.click(screen.getByRole('button', { name: '確認してログイン' }))
    await vi.waitFor(() => expect(session.signIn).toHaveBeenCalledWith(me))
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/auth/login/recheck/verify',
      expect.objectContaining({
        body: JSON.stringify({ challengeToken: 'challenge-token', code: '123456' }),
      }),
    )
  })

  it('違うコードはサーバーの文言を出し、入力を消す', async () => {
    const { user, session } = setup(() =>
      json(
        {
          code: 'invalid_verification_code',
          message: 'コードが違います。あと3回間違えるとこのコードは使えなくなります。',
        },
        400,
      ),
    )
    await enterCode(user, '000000')
    await user.click(screen.getByRole('button', { name: '確認してログイン' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('あと3回間違えると')
    expect(screen.getByLabelText('1桁目')).toHaveValue('')
    expect(session.signIn).not.toHaveBeenCalled()
  })

  it('チャレンジが切れていたら、ログイン画面に戻る導線を出す', async () => {
    const { user } = setup(() =>
      json({ code: 'invalid_login_challenge', message: 'もう一度ログインしてください' }, 400),
    )
    await enterCode(user, '123456')
    await user.click(screen.getByRole('button', { name: '確認してログイン' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('もう一度ログインしてください')
    expect(screen.getByRole('link', { name: 'ログイン画面に戻る' })).toHaveAttribute(
      'href',
      '/login',
    )
    expect(screen.getByRole('button', { name: '確認してログイン' })).toBeDisabled()
  })

  it('1分たったらチャレンジでコードを送り直し、開こうとしていたページを残す', async () => {
    const { user, router, fetchMock } = setup(() => new Response(null, { status: 202 }), {
      challenge: challenge(),
      sentAt: Date.now(),
      from: '/deals/42',
    })
    act(() => vi.advanceTimersByTime(60_000))
    await user.click(screen.getByRole('button', { name: 'コードを再送' }))
    expect(await screen.findByRole('button', { name: 'コードを再送（1:00 後）' })).toBeDisabled()
    expect(screen.getByText('10:00')).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/auth/login/recheck/codes',
      expect.objectContaining({ body: JSON.stringify({ challengeToken: 'challenge-token' }) }),
    )
    expect(router.state.location.state).toMatchObject({ from: '/deals/42' })
  })

  it('ログイン画面へ戻るときも、開こうとしていたページを引き継ぐ', async () => {
    const { user, router } = setup(() => json({}), {
      challenge: challenge(),
      sentAt: Date.now(),
      from: '/deals/42',
    })
    await user.click(screen.getByRole('link', { name: '別のアカウント' }))
    expect(router.state.location.pathname).toBe('/login')
    expect(router.state.location.state).toEqual({ from: '/deals/42' })
  })
})
