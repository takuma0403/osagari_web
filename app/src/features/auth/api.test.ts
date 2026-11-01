import { getMe, login, sendPasswordResetCode, sendSignupCode, verifyRecheck } from './api'

const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })

describe('auth api', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('学籍番号から大学のメールアドレスを作って JSON で送る', async () => {
    const fetchMock = vi.fn(async () => new Response(null, { status: 202 }))
    vi.stubGlobal('fetch', fetchMock)
    await sendSignupCode('26238001')
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/auth/signup/codes',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ email: '26238001@edu.cc.saga-u.ac.jp' }),
        headers: expect.objectContaining({ 'Content-Type': 'application/json' }),
      }),
    )
  })

  it('ログインの 200 はログイン済みとして返す', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => json({ id: 'u1', email: '26238001@edu.cc.saga-u.ac.jp' })),
    )
    await expect(login('26238001', 'password12')).resolves.toMatchObject({
      kind: 'loggedIn',
      user: { id: 'u1' },
    })
  })

  it('ログインの 202 は再確認のチャレンジとして返す', async () => {
    const challenge = {
      challengeToken: 't',
      reason: 'new_device',
      email: '26238001@edu.cc.saga-u.ac.jp',
    }
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => json(challenge, 202)),
    )
    await expect(login('26238001', 'password12')).resolves.toEqual({
      kind: 'recheck',
      challenge,
    })
  })

  it('パスワード再設定のコードも学籍番号からアドレスを作って送る', async () => {
    const fetchMock = vi.fn(async () => new Response(null, { status: 202 }))
    vi.stubGlobal('fetch', fetchMock)
    await sendPasswordResetCode('26238001')
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/auth/password-reset/codes',
      expect.objectContaining({ body: JSON.stringify({ email: '26238001@edu.cc.saga-u.ac.jp' }) }),
    )
  })

  it('再確認はチャレンジのトークンとコードだけを送る（パスワードは送らない）', async () => {
    const fetchMock = vi.fn(async () => json({ id: 'u1' }))
    vi.stubGlobal('fetch', fetchMock)
    await verifyRecheck('challenge', '123456')
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/auth/login/recheck/verify',
      expect.objectContaining({
        body: JSON.stringify({ challengeToken: 'challenge', code: '123456' }),
      }),
    )
  })

  it('本人の情報は GET /me で取る', async () => {
    const fetchMock = vi.fn(async () => json({ id: 'u1' }))
    vi.stubGlobal('fetch', fetchMock)
    await expect(getMe()).resolves.toEqual({ id: 'u1' })
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/me',
      expect.not.objectContaining({ method: 'POST' }),
    )
  })
})
