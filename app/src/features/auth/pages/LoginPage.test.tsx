import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderRoute } from '../../../test/renderRoute'
import { LoginPage } from './LoginPage'

const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })

function setup(response: () => Response) {
  const fetchMock = vi.fn(async () => response())
  vi.stubGlobal('fetch', fetchMock)
  const user = userEvent.setup()
  const view = renderRoute('/login', <LoginPage />, { destinations: ['/', '/recheck'] })
  return { ...view, user, fetchMock }
}

async function fillAndSubmit(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText('大学のメールアドレス'), '26238001')
  await user.type(screen.getByLabelText('パスワード'), 'password12')
  await user.click(screen.getByRole('button', { name: 'ログイン' }))
}

describe('LoginPage', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('学籍番号とパスワードがそろうまでログインを押せない', async () => {
    const { user } = setup(() => json({}))
    const submit = screen.getByRole('button', { name: 'ログイン' })
    expect(submit).toBeDisabled()
    await user.type(screen.getByLabelText('大学のメールアドレス'), '26238001')
    expect(submit).toBeDisabled()
    await user.type(screen.getByLabelText('パスワード'), 'p')
    expect(submit).toBeEnabled()
  })

  it('ログインできたら商品一覧へ進む', async () => {
    const { user, router, fetchMock } = setup(() => json({ id: 'u1' }))
    await fillAndSubmit(user)
    expect(await screen.findByRole('heading', { name: '/' })).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/')
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/auth/login',
      expect.objectContaining({
        body: JSON.stringify({ email: '26238001@edu.cc.saga-u.ac.jp', password: 'password12' }),
        headers: expect.objectContaining({ 'X-Device-Id': expect.any(String) }),
      }),
    )
  })

  it('再確認が必要（202）なら、チャレンジを持って /recheck へ進む', async () => {
    const challenge = {
      challengeToken: 't',
      reason: 'new_device',
      email: '26238001@edu.cc.saga-u.ac.jp',
    }
    const { user } = setup(() => json(challenge, 202))
    await fillAndSubmit(user)
    const heading = await screen.findByRole('heading', { name: '/recheck' })
    expect(JSON.parse(heading.dataset.state!)).toEqual({ challenge })
  })

  it('失敗したらサーバーの文言を出し、入力を直すと消す', async () => {
    const { user } = setup(() =>
      json({ code: 'invalid_credentials', message: 'メールアドレスかパスワードが違います' }, 401),
    )
    await fillAndSubmit(user)
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'メールアドレスかパスワードが違います',
    )
    await user.type(screen.getByLabelText('パスワード'), 'x')
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('ログインが止まっている（429）ときは再設定を案内し、ボタンを押せなくする', async () => {
    const { user } = setup(() =>
      json(
        {
          code: 'too_many_requests',
          message: 'ログインの失敗が続いたため、しばらくログインできません',
        },
        429,
      ),
    )
    await fillAndSubmit(user)
    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('しばらくログインできません')
    expect(alert).toHaveTextContent('思い出せない場合は、パスワードを再設定してください。')
    expect(screen.getByRole('button', { name: 'ログイン' })).toBeDisabled()
  })

  it('新規登録・パスワード再設定・規約へのリンクがある', () => {
    setup(() => json({}))
    expect(screen.getByRole('link', { name: '新規登録' })).toHaveAttribute('href', '/signup')
    expect(screen.getByRole('link', { name: 'パスワードを忘れた場合' })).toHaveAttribute(
      'href',
      '/password-reset',
    )
    expect(screen.getByRole('link', { name: '利用規約' })).toHaveAttribute('href', '/terms')
    expect(screen.getByRole('link', { name: 'プライバシーポリシー' })).toHaveAttribute(
      'href',
      '/privacy',
    )
  })
})
