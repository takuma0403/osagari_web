import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderRoute } from '../../../test/renderRoute'
import { SignUpPage } from './SignUpPage'

const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })

const estimate = (overrides: Record<string, unknown> = {}) => ({
  studentNumber: '26238001',
  admissionYear: 2026,
  departmentCode: '238',
  department: '238',
  departmentName: '理工学部 理工学科',
  expiresOn: '2030-03-31',
  ...overrides,
})

type Handler = (path: string) => Response

function setup(handler: Handler) {
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => handler(String(input)))
  vi.stubGlobal('fetch', fetchMock)
  const user = userEvent.setup()
  const view = renderRoute('/signup', <SignUpPage />, { destinations: ['/signup/code'] })
  return { ...view, user, fetchMock }
}

const ok: Handler = (path) =>
  path.endsWith('/estimate') ? json(estimate()) : new Response(null, { status: 202 })

describe('SignUpPage', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('8桁そろったら学部・入学年度・有効期限の推定を出す', async () => {
    const { user } = setup(ok)
    await user.type(screen.getByLabelText('大学のメールアドレス'), '2623800')
    expect(screen.queryByText(/として登録されます/)).not.toBeInTheDocument()
    await user.type(screen.getByLabelText('大学のメールアドレス'), '1')
    expect(
      await screen.findByText(
        '理工学部 理工学科・2026年度入学として登録されます（有効期限 2030年3月31日）',
      ),
    ).toBeInTheDocument()
  })

  it('表にない学部コードは「学部不明」と出す', async () => {
    const { user } = setup(() =>
      json(estimate({ department: null, departmentName: null, departmentCode: '999' })),
    )
    await user.type(screen.getByLabelText('大学のメールアドレス'), '26999001')
    expect(await screen.findByText(/^学部不明・2026年度入学/)).toBeInTheDocument()
  })

  it('入力を変えたら推定を消す', async () => {
    const { user } = setup(ok)
    const input = screen.getByLabelText('大学のメールアドレス')
    await user.type(input, '26238001')
    await screen.findByText(/として登録されます/)
    await user.type(input, '{Backspace}')
    expect(screen.queryByText(/として登録されます/)).not.toBeInTheDocument()
  })

  it('コードを送ったら、学籍番号と送った時刻を持って /signup/code へ進む', async () => {
    const { user, fetchMock } = setup(ok)
    const submit = screen.getByRole('button', { name: '認証コードを送る' })
    expect(submit).toBeDisabled()
    await user.type(screen.getByLabelText('大学のメールアドレス'), '26238001')
    await user.click(submit)
    const heading = await screen.findByRole('heading', { name: '/signup/code' })
    expect(JSON.parse(heading.dataset.state!)).toEqual({
      studentId: '26238001',
      sentAt: expect.any(Number),
    })
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/auth/signup/codes',
      expect.objectContaining({ body: JSON.stringify({ email: '26238001@edu.cc.saga-u.ac.jp' }) }),
    )
  })

  it('送れなかったときはサーバーの文言を出す', async () => {
    const { user } = setup((path) =>
      path.endsWith('/estimate')
        ? json(estimate())
        : json(
            {
              code: 'too_many_requests',
              message: 'コードを送ったばかりです。1分ほどたってからもう一度お試しください',
            },
            429,
          ),
    )
    await user.type(screen.getByLabelText('大学のメールアドレス'), '26238001')
    await user.click(screen.getByRole('button', { name: '認証コードを送る' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('コードを送ったばかりです')
    expect(screen.getByRole('button', { name: '認証コードを送る' })).toBeEnabled()
  })

  it('戻るとログインへのリンクがある', () => {
    setup(ok)
    expect(screen.getByRole('link', { name: '戻る' })).toHaveAttribute('href', '/login')
    expect(screen.getByRole('link', { name: 'ログイン' })).toHaveAttribute('href', '/login')
  })
})
