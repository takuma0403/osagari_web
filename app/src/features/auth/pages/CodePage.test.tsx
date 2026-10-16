import { act, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderRoute } from '../../../test/renderRoute'
import { CodePage } from './CodePage'

const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })

const verified = {
  setupToken: 'setup-token',
  estimate: {
    studentNumber: '26238001',
    admissionYear: 2026,
    departmentCode: '238',
    department: '238',
    departmentName: '理工学部 理工学科',
    expiresOn: '2030-03-31',
  },
}

function setup(handler: (path: string) => Response, state: unknown = undefined) {
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => handler(String(input)))
  vi.stubGlobal('fetch', fetchMock)
  const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })
  const view = renderRoute('/signup/code', <CodePage flow="signup" />, {
    state: state === undefined ? { studentId: '26238001', sentAt: Date.now() } : state,
    destinations: ['/signup', '/signup/password'],
  })
  return { ...view, user, fetchMock }
}

async function enterCode(user: ReturnType<typeof userEvent.setup>, code: string) {
  await user.click(screen.getByLabelText('1桁目'))
  await user.paste(code)
}

describe('CodePage（新規登録）', () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    vi.setSystemTime(new Date('2026-10-16T12:00:00+09:00'))
  })
  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('送った学籍番号がなければ /signup へ戻す', () => {
    const { router } = setup(() => json({}), null)
    expect(router.state.location.pathname).toBe('/signup')
  })

  it('送り先のアドレス・有効期限・再送までの待ち時間を出す', () => {
    setup(() => json({}))
    expect(screen.getByRole('heading', { level: 1, name: '認証コードを入力' })).toBeInTheDocument()
    expect(screen.getByText('STEP 2 / 3')).toBeInTheDocument()
    expect(screen.getByText('26238001@edu.cc.saga-u.ac.jp')).toBeInTheDocument()
    expect(screen.getByText('10:00')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'コードを再送（1:00 後）' })).toBeDisabled()
  })

  it('6桁を確認できたら、一時トークンを持って /signup/password へ進む', async () => {
    const { user, fetchMock } = setup(() => json(verified))
    const submit = screen.getByRole('button', { name: '確認する' })
    expect(submit).toBeDisabled()
    await enterCode(user, '123456')
    await user.click(submit)
    const heading = await screen.findByRole('heading', { name: '/signup/password' })
    expect(JSON.parse(heading.dataset.state!)).toEqual(verified)
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/auth/signup/codes/verify',
      expect.objectContaining({
        body: JSON.stringify({ email: '26238001@edu.cc.saga-u.ac.jp', code: '123456' }),
      }),
    )
  })

  it('違うコードはサーバーの文言（残り回数）を出し、入力を消す', async () => {
    const { user } = setup(() =>
      json(
        {
          code: 'invalid_verification_code',
          message: 'コードが違います。あと3回間違えるとこのコードは使えなくなります。',
        },
        400,
      ),
    )
    await enterCode(user, '000000')
    await user.click(screen.getByRole('button', { name: '確認する' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('あと3回間違えると')
    expect(screen.getByLabelText('1桁目')).toHaveValue('')
  })

  it('1分たったらコードを送り直せ、待ち時間を数え直す', async () => {
    const { user, fetchMock } = setup(() => new Response(null, { status: 202 }))
    act(() => vi.advanceTimersByTime(60_000))
    const resend = screen.getByRole('button', { name: 'コードを再送' })
    expect(resend).toBeEnabled()
    await user.click(resend)
    expect(await screen.findByRole('button', { name: 'コードを再送（1:00 後）' })).toBeDisabled()
    expect(screen.getByText('10:00')).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/auth/signup/codes',
      expect.objectContaining({ body: JSON.stringify({ email: '26238001@edu.cc.saga-u.ac.jp' }) }),
    )
  })

  it('送り直せなかった（429）ときはサーバーの文言を出す', async () => {
    const { user } = setup(() =>
      json(
        {
          code: 'too_many_requests',
          message:
            'コードの送信が続いたため、しばらく送れません。1時間ほどたってからもう一度お試しください',
        },
        429,
      ),
    )
    act(() => vi.advanceTimersByTime(60_000))
    await user.click(screen.getByRole('button', { name: 'コードを再送' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('しばらく送れません')
  })

  it('登録済みの案内とログインへのリンクを出す', () => {
    setup(() => json({}))
    expect(screen.getByText(/「登録済みです」という案内が届きます/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'ログイン' })).toHaveAttribute('href', '/login')
    expect(screen.getByRole('link', { name: '戻る' })).toHaveAttribute('href', '/signup')
  })
})
