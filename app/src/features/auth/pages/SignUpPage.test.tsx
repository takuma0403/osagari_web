import { screen, within } from '@testing-library/react'
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

type User = ReturnType<typeof userEvent.setup>

/** 規約の文書をシートで開いて閉じる */
async function openDoc(user: User, name: '利用規約' | 'プライバシーポリシー') {
  await user.click(screen.getByRole('button', { name }))
  const sheet = await screen.findByRole('dialog', { name })
  expect(await within(sheet).findAllByRole('heading', { level: 2 })).not.toHaveLength(0)
  await user.click(within(sheet).getByRole('button', { name: '閉じる' }))
}

/** 両方の文書を開いて、同意にチェックする */
async function agree(user: User) {
  await openDoc(user, '利用規約')
  await openDoc(user, 'プライバシーポリシー')
  await user.click(screen.getByRole('checkbox', { name: /同意します/ }))
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
    expect(submit).toBeDisabled()
    await agree(user)
    expect(submit).toBeEnabled()
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
    await agree(user)
    await user.click(screen.getByRole('button', { name: '認証コードを送る' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('コードを送ったばかりです')
    expect(screen.getByRole('button', { name: '認証コードを送る' })).toBeEnabled()
  })

  it('戻るとログインへのリンクがある', () => {
    setup(ok)
    expect(screen.getByRole('link', { name: '戻る' })).toHaveAttribute('href', '/login')
    expect(screen.getByRole('link', { name: 'ログイン' })).toHaveAttribute('href', '/login')
  })

  it('利用規約とプライバシーポリシーの両方を開くまで、同意にチェックできない', async () => {
    const { user } = setup(ok)
    const checkbox = screen.getByRole('checkbox', { name: /同意します/ })
    expect(checkbox).toBeDisabled()
    expect(checkbox).toHaveAccessibleDescription('両方の文書を開くとチェックできます')
    await openDoc(user, '利用規約')
    expect(checkbox).toBeDisabled()
    await openDoc(user, 'プライバシーポリシー')
    expect(checkbox).toBeEnabled()
    expect(screen.queryByText('両方の文書を開くとチェックできます')).not.toBeInTheDocument()
  })

  it('同意のチェックを外したらコードを送れない', async () => {
    const { user } = setup(ok)
    await user.type(screen.getByLabelText('大学のメールアドレス'), '26238001')
    await agree(user)
    const submit = screen.getByRole('button', { name: '認証コードを送る' })
    expect(submit).toBeEnabled()
    await user.click(screen.getByRole('checkbox', { name: /同意します/ }))
    expect(submit).toBeDisabled()
  })

  it('規約の本文中のプライバシーポリシーへのリンクは、画面を移らずにシートの文書を切り替える', async () => {
    const { user } = setup(ok)
    await user.type(screen.getByLabelText('大学のメールアドレス'), '26238001')
    await user.click(screen.getByRole('button', { name: '利用規約' }))
    const sheet = await screen.findByRole('dialog', { name: '利用規約' })
    await user.click(await within(sheet).findByRole('link', { name: 'プライバシーポリシー' }))
    const privacy = await screen.findByRole('dialog', { name: 'プライバシーポリシー' })
    await user.click(within(privacy).getByRole('button', { name: '閉じる' }))
    expect(screen.getByLabelText('大学のメールアドレス')).toHaveValue('26238001')
    // 本文のリンクから開いたプライバシーポリシーも、開いたものとして数える
    expect(screen.getByRole('checkbox', { name: /同意します/ })).toBeEnabled()
  })
})

describe('SignUpPage（パスワード再設定）', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  function setupReset() {
    const fetchMock = vi.fn(async () => new Response(null, { status: 202 }))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    const view = renderRoute('/password-reset', <SignUpPage flow="passwordReset" />, {
      destinations: ['/password-reset/code'],
    })
    return { ...view, user, fetchMock }
  }

  it('再設定の見出しと説明を出し、推定・規約への同意・「登録済みの方は」は出さない', async () => {
    const { user, fetchMock } = setupReset()
    expect(
      screen.getByRole('heading', { level: 1, name: 'パスワードの再設定' }),
    ).toBeInTheDocument()
    expect(screen.getByText(/新しいパスワードを決めます/)).toBeInTheDocument()
    await user.type(screen.getByLabelText('大学のメールアドレス'), '26238001')
    expect(screen.queryByText(/登録済みの方は/)).not.toBeInTheDocument()
    expect(screen.queryByRole('checkbox')).not.toBeInTheDocument()
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('コードを送ったら、学籍番号と送った時刻を持って /password-reset/code へ進む', async () => {
    const { user, fetchMock } = setupReset()
    await user.type(screen.getByLabelText('大学のメールアドレス'), '26238001')
    await user.click(screen.getByRole('button', { name: '認証コードを送る' }))
    const heading = await screen.findByRole('heading', { name: '/password-reset/code' })
    expect(JSON.parse(heading.dataset.state!)).toMatchObject({ studentId: '26238001' })
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/auth/password-reset/codes',
      expect.objectContaining({ body: JSON.stringify({ email: '26238001@edu.cc.saga-u.ac.jp' }) }),
    )
  })
})
