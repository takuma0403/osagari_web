import { login, sendSignupCode } from './api'

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
})
