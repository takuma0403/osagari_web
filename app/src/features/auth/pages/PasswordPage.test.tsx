import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderRoute } from '../../../test/renderRoute'
import { PasswordPage } from './PasswordPage'

const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })

const me = { id: 'u1', email: '26238001@edu.cc.saga-u.ac.jp' }

const signupState = {
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

function setup(
  flow: 'signup' | 'passwordReset',
  response: () => Response,
  state: unknown = flow === 'signup' ? signupState : { setupToken: 'reset-token' },
) {
  const fetchMock = vi.fn(async () => response())
  vi.stubGlobal('fetch', fetchMock)
  const user = userEvent.setup()
  const path = flow === 'signup' ? '/signup/password' : '/password-reset/password'
  const view = renderRoute(path, <PasswordPage flow={flow} />, {
    state,
    destinations: ['/signup', '/password-reset'],
  })
  return { ...view, user, fetchMock }
}

async function fill(
  user: ReturnType<typeof userEvent.setup>,
  password: string,
  confirm = password,
) {
  await user.type(screen.getByLabelText('パスワード'), password)
  await user.type(screen.getByLabelText('パスワード（確認）'), confirm)
}

describe('PasswordPage（新規登録）', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('一時トークンがなければ /signup へ戻す', () => {
    const { router } = setup('signup', () => json(me), null)
    expect(router.state.location.pathname).toBe('/signup')
  })

  it('見出し・STEP・登録される内容を出す', () => {
    setup('signup', () => json(me))
    expect(
      screen.getByRole('heading', { level: 1, name: 'パスワードを決める' }),
    ).toBeInTheDocument()
    expect(screen.getByText('STEP 3 / 3')).toBeInTheDocument()
    expect(screen.getByText('26238001')).toBeInTheDocument()
    expect(screen.getByText('理工学部 理工学科')).toBeInTheDocument()
    expect(screen.getByText('2030年3月31日')).toBeInTheDocument()
    expect(screen.queryByText(/ほかの端末ではログアウト/)).not.toBeInTheDocument()
  })

  it('学部が不明なら「学部不明」と出す', () => {
    setup('signup', () => json(me), {
      ...signupState,
      estimate: { ...signupState.estimate, department: null, departmentName: null },
    })
    expect(screen.getByText('学部不明')).toBeInTheDocument()
  })

  it('10文字そろうまで押せず、文字数を出す', async () => {
    const { user } = setup('signup', () => json(me))
    const submit = screen.getByRole('button', { name: '登録してはじめる' })
    await fill(user, 'abcdefghi')
    expect(screen.getByText('10文字以上（9文字）')).toBeInTheDocument()
    expect(submit).toBeDisabled()
    await user.type(screen.getByLabelText('パスワード'), 'j')
    expect(screen.getByText('10文字以上（10文字）')).toBeInTheDocument()
    expect(submit).toBeEnabled()
  })

  it('文字数は文字単位で数える（AUTHN-5）', async () => {
    const { user } = setup('signup', () => json(me))
    await user.type(screen.getByLabelText('パスワード'), '🍣🍣🍣🍣🍣')
    expect(screen.getByText('10文字以上（5文字）')).toBeInTheDocument()
  })

  it('確認が一致しなければ送らずに伝える', async () => {
    const { user, fetchMock } = setup('signup', () => json(me))
    await fill(user, 'sagaunifurima', 'sagaunifurimb')
    await user.click(screen.getByRole('button', { name: '登録してはじめる' }))
    expect(screen.getByRole('alert')).toHaveTextContent('パスワードが一致しません')
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('128文字を超えたら伝えて押せなくする', async () => {
    const { user } = setup('signup', () => json(me))
    await user.click(screen.getByLabelText('パスワード'))
    await user.paste('a'.repeat(129))
    expect(screen.getByRole('alert')).toHaveTextContent('128文字以下にしてください（129文字）')
  })

  it('設定できたらログイン状態にする', async () => {
    const { user, session, fetchMock } = setup('signup', () => json(me, 201))
    await fill(user, 'sagaunifurima')
    await user.click(screen.getByRole('button', { name: '登録してはじめる' }))
    await vi.waitFor(() => expect(session.signIn).toHaveBeenCalledWith(me))
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/auth/signup/complete',
      expect.objectContaining({
        body: JSON.stringify({ setupToken: 'setup-token', password: 'sagaunifurima' }),
      }),
    )
  })

  it('一時トークンが切れていたら、サーバーの文言とやり直しのリンクを出す', async () => {
    const { user, session } = setup('signup', () =>
      json({ code: 'invalid_setup_token', message: '有効期限が切れました' }, 400),
    )
    await fill(user, 'sagaunifurima')
    await user.click(screen.getByRole('button', { name: '登録してはじめる' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('有効期限が切れました')
    expect(screen.getByRole('link', { name: '最初からやり直す' })).toHaveAttribute(
      'href',
      '/signup',
    )
    expect(session.signIn).not.toHaveBeenCalled()
  })
})

describe('PasswordPage（パスワード再設定）', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('一時トークンがなければ /password-reset へ戻す', () => {
    const { router } = setup('passwordReset', () => json(me), null)
    expect(router.state.location.pathname).toBe('/password-reset')
  })

  it('見出しとほかの端末がログアウトされる注意を出し、登録内容は出さない', () => {
    setup('passwordReset', () => json(me))
    expect(screen.getByRole('heading', { level: 1, name: '新しいパスワード' })).toBeInTheDocument()
    expect(screen.getByText('変更すると、ほかの端末ではログアウトされます。')).toBeInTheDocument()
    expect(screen.queryByText('学籍番号')).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: '戻る' })).toHaveAttribute('href', '/password-reset')
  })

  it('変更できたらログイン状態にする', async () => {
    const { user, session, fetchMock } = setup('passwordReset', () => json(me))
    await fill(user, 'sagaunifurima')
    await user.click(screen.getByRole('button', { name: 'パスワードを変更する' }))
    await vi.waitFor(() => expect(session.signIn).toHaveBeenCalledWith(me))
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/auth/password-reset/complete',
      expect.objectContaining({
        body: JSON.stringify({ setupToken: 'reset-token', password: 'sagaunifurima' }),
      }),
    )
  })

  it('停止中・失効のアカウントはサーバーの文言を出す', async () => {
    const { user } = setup('passwordReset', () =>
      json({ code: 'account_suspended', message: 'このアカウントは停止されています' }, 403),
    )
    await fill(user, 'sagaunifurima')
    await user.click(screen.getByRole('button', { name: 'パスワードを変更する' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('このアカウントは停止されています')
    expect(screen.queryByRole('link', { name: '最初からやり直す' })).not.toBeInTheDocument()
  })
})
