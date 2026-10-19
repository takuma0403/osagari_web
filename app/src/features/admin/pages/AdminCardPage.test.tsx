import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { AdminLayout } from '../../../layouts/AdminLayout/AdminLayout'
import type { CurrentUser } from '../../auth/api'
import { SessionContext, type Session } from '../../auth/session'
import type { AdminStudentCard, StudentCardView } from '../api'
import { AdminCardPage } from './AdminCardPage'

const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })

const card = (id: string, studentNumber: string, departmentName: string | null) =>
  ({
    id,
    status: 'pending',
    studentNumber,
    departmentName,
    admissionYear: 2026,
    expiresOn: '2030-03-31',
    submittedAt: '2026-10-06T01:12:00Z',
    reviewedAt: null,
  }) satisfies AdminStudentCard

const first = card('c1', '26141023', '経済学部 経済学科')
const second = card('c2', '72612004', null)
const me = { id: 'admin', studentNumber: '99999999', sellerStatus: 'approved' } as CurrentUser

const viewOf = (adminStudentNumber: string): StudentCardView => ({
  viewedAt: '2026-10-08T06:20:00Z',
  adminStudentNumber,
})

type Routes = {
  list?: () => Response
  view?: (id: string) => Response
  views?: (id: string) => Response
  approve?: (id: string) => Response
  reject?: (id: string) => Response
}

/** URL とメソッドで応答を返し分ける fetch を用意し、管理画面の枠の中に画面を開く */
function setup(routes: Routes = {}, user: CurrentUser = me) {
  const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
    const method = init?.method ?? 'GET'
    if (url === '/api/admin/student-cards') return (routes.list ?? (() => json([first, second])))()
    const m = url.match(/^\/api\/admin\/student-cards\/([^/]+)\/(views|approve|reject)$/)
    if (m) {
      const [, id, action] = m
      if (action === 'views' && method === 'POST')
        return (
          routes.view ??
          ((i) => json({ url: `https://storage.test/${i}.webp`, expiresAt: '', viewedAt: '' }, 201))
        )(id)
      if (action === 'views')
        return (routes.views ?? (() => json([viewOf('99999999'), viewOf('11111111')])))(id)
      if (action === 'approve')
        return (
          routes.approve ?? ((i) => json({ ...(i === 'c1' ? first : second), status: 'approved' }))
        )(id)
      if (action === 'reject')
        return (
          routes.reject ?? ((i) => json({ ...(i === 'c1' ? first : second), status: 'rejected' }))
        )(id)
    }
    throw new Error(`想定していない呼び出し: ${method} ${url}`)
  })
  vi.stubGlobal('fetch', fetchMock)
  const router = createMemoryRouter(
    [
      {
        element: <AdminLayout />,
        children: [{ path: '/admin/student-cards', element: <AdminCardPage /> }],
      },
    ],
    { initialEntries: ['/admin/student-cards'] },
  )
  const session: Session = { user, signIn: vi.fn(), signOut: vi.fn() }
  render(
    <SessionContext value={session}>
      <RouterProvider router={router} />
    </SessionContext>,
  )
  return { fetchMock, session, user: userEvent.setup() }
}

const calls = (fetchMock: ReturnType<typeof setup>['fetchMock'], method: string, url: string) =>
  fetchMock.mock.calls.filter(([u, init]) => u === url && (init?.method ?? 'GET') === method)

async function enterCardNumber(user: ReturnType<typeof userEvent.setup>, value: string) {
  const input = await screen.findByLabelText('学生証に書かれた番号')
  await user.clear(input)
  await user.type(input, value)
}

describe('AdminCardPage', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('審査待ちを古い順に出し、件数を見出しとメニューに出す', async () => {
    setup()
    expect(screen.getByText('読み込み中…')).toHaveAttribute('role', 'status')
    expect(await screen.findByText('審査待ち 2件 · 古い順')).toBeInTheDocument()
    const queue = screen.getByRole('region', { name: '審査待ち一覧' })
    const rows = within(queue).getAllByRole('button')
    expect(rows).toHaveLength(2)
    expect(rows[0]).toHaveTextContent('26141023')
    expect(rows[0]).toHaveTextContent('経済学部 経済学科')
    expect(rows[1]).toHaveTextContent('不明（表にないコード）')
    expect(await screen.findByRole('link', { name: '学生証審査 審査待ち2件' })).toBeInTheDocument()
  })

  it('先頭を選び、画像の閲覧URLを発行してから閲覧記録を出す', async () => {
    const { fetchMock } = setup()
    expect(await screen.findByRole('img', { name: '学生証の画像（26141023）' })).toHaveAttribute(
      'src',
      'https://storage.test/c1.webp',
    )
    const queue = screen.getByRole('region', { name: '審査待ち一覧' })
    expect(within(queue).getAllByRole('button')[0]).toHaveAttribute('aria-pressed', 'true')
    const panel = screen.getByRole('region', { name: '審査' })
    expect(within(panel).getByText('2026年度')).toBeInTheDocument()
    expect(within(panel).getByText('2030年3月31日')).toBeInTheDocument()
    expect(
      await within(panel).findByText(/この画像の閲覧は記録されます（.* · 99999999）/),
    ).toBeInTheDocument()
    expect(within(panel).getByText('閲覧記録を見る（2件）')).toBeInTheDocument()
    expect(calls(fetchMock, 'POST', '/api/admin/student-cards/c1/views')).toHaveLength(1)
  })

  it('別の行を選ぶと、その学生証の閲覧URLを発行する', async () => {
    const { fetchMock, user } = setup()
    await screen.findByRole('img', { name: '学生証の画像（26141023）' })
    const queue = screen.getByRole('region', { name: '審査待ち一覧' })
    await user.click(within(queue).getAllByRole('button')[1])
    expect(await screen.findByRole('img', { name: '学生証の画像（72612004）' })).toBeInTheDocument()
    expect(calls(fetchMock, 'POST', '/api/admin/student-cards/c2/views')).toHaveLength(1)
  })

  it('番号がメールの学籍番号と一致したときだけ承認できる', async () => {
    const { user } = setup()
    const approve = await screen.findByRole('button', { name: '承認する' })
    expect(approve).toBeDisabled()

    await enterCardNumber(user, '26141024')
    expect(screen.getByText('メールの学籍番号と一致しません')).toBeInTheDocument()
    expect(approve).toBeDisabled()

    await enterCardNumber(user, '２６１４１０２３')
    expect(screen.getByText('メールの学籍番号と一致')).toBeInTheDocument()
    expect(approve).toBeEnabled()
  })

  it('承認は確認してから行い、一覧から外して次の学生証を選ぶ', async () => {
    const { fetchMock, user } = setup()
    await enterCardNumber(user, '26141023')
    await user.click(screen.getByRole('button', { name: '承認する' }))

    const dialog = screen.getByRole('dialog', { name: '学生証を承認しますか？' })
    expect(calls(fetchMock, 'POST', '/api/admin/student-cards/c1/approve')).toHaveLength(0)
    await user.click(within(dialog).getByRole('button', { name: '承認する' }))

    expect(await screen.findByText('26141023 を承認しました')).toBeInTheDocument()
    expect(calls(fetchMock, 'POST', '/api/admin/student-cards/c1/approve')).toHaveLength(1)
    expect(screen.getByText('審査待ち 1件 · 古い順')).toBeInTheDocument()
    expect(await screen.findByRole('img', { name: '学生証の画像（72612004）' })).toBeInTheDocument()
    // 次の学生証では入力した番号を消す
    expect(screen.getByLabelText('学生証に書かれた番号')).toHaveValue('')
  })

  it('確認でやめると承認しない', async () => {
    const { fetchMock, user } = setup()
    await enterCardNumber(user, '26141023')
    await user.click(screen.getByRole('button', { name: '承認する' }))
    await user.click(screen.getByRole('button', { name: 'やめる' }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(calls(fetchMock, 'POST', '/api/admin/student-cards/c1/approve')).toHaveLength(0)
  })

  it('差し戻しは番号を入れなくても確認してから行える', async () => {
    const { fetchMock, user } = setup()
    await user.click(await screen.findByRole('button', { name: '差し戻す' }))
    const dialog = screen.getByRole('dialog', { name: '学生証を差し戻しますか？' })
    await user.click(within(dialog).getByRole('button', { name: '差し戻す' }))
    expect(await screen.findByText('26141023 を差し戻しました')).toBeInTheDocument()
    expect(calls(fetchMock, 'POST', '/api/admin/student-cards/c1/reject')).toHaveLength(1)
  })

  it('最後の1件を審査すると、審査待ちがないことを出す', async () => {
    const { user } = setup({ list: () => json([second]) })
    await user.click(await screen.findByRole('button', { name: '差し戻す' }))
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: '差し戻す' }))
    expect(await screen.findByText('審査待ちの学生証はありません。')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: '学生証審査' })).toBeInTheDocument()
  })

  it('自分の学生証を審査したら、セッションの出品者の状態も揃える', async () => {
    const self = { ...me, studentNumber: '26141023', sellerStatus: 'pending' } as CurrentUser
    const { session, user } = setup({}, self)
    await enterCardNumber(user, '26141023')
    await user.click(screen.getByRole('button', { name: '承認する' }))
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: '承認する' }))
    await waitFor(() =>
      expect(session.signIn).toHaveBeenCalledWith({ ...self, sellerStatus: 'approved' }),
    )
  })

  it('状態が変わっていた（409）ときは知らせて一覧を取り直す', async () => {
    let listed = 0
    const { user } = setup({
      list: () => json(listed++ === 0 ? [first, second] : [second]),
      reject: () =>
        json(
          { code: 'invalid_status_transition', message: '状態が変わったため操作できませんでした' },
          409,
        ),
    })
    await user.click(await screen.findByRole('button', { name: '差し戻す' }))
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: '差し戻す' }))
    expect(await screen.findByText('状態が変わったため操作できませんでした')).toBeInTheDocument()
    expect(await screen.findByText('審査待ち 1件 · 古い順')).toBeInTheDocument()
  })

  it('承認に失敗したらエラーを出し、一覧はそのままにする', async () => {
    const { user } = setup({
      approve: () =>
        json({ code: 'internal_error', message: 'サーバーでエラーが発生しました' }, 500),
    })
    await enterCardNumber(user, '26141023')
    await user.click(screen.getByRole('button', { name: '承認する' }))
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: '承認する' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('サーバーでエラーが発生しました')
    expect(screen.getByText('審査待ち 2件 · 古い順')).toBeInTheDocument()
  })

  it('画像の閲覧URLを発行できなければ、発行し直せる', async () => {
    let issued = 0
    const { user } = setup({
      view: (id) =>
        issued++ === 0
          ? json({ code: 'not_found', message: '見つかりません' }, 404)
          : json({ url: `https://storage.test/${id}.webp`, expiresAt: '', viewedAt: '' }, 201),
    })
    expect(await screen.findByText('見つかりません')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'もう一度表示する' }))
    expect(await screen.findByRole('img', { name: '学生証の画像（26141023）' })).toBeInTheDocument()
  })

  it('審査待ちがなければその旨を出す', async () => {
    setup({ list: () => json([]) })
    expect(await screen.findByText('審査待ちの学生証はありません。')).toBeInTheDocument()
    expect(screen.getByText('審査待ち 0件 · 古い順')).toBeInTheDocument()
  })

  it('一覧を読み込めなければエラーを出し、読み込み直せる', async () => {
    let listed = 0
    const { user } = setup({
      list: () =>
        listed++ === 0
          ? json({ code: 'internal_error', message: 'サーバーでエラーが発生しました' }, 500)
          : json([first]),
    })
    expect(await screen.findByRole('alert')).toHaveTextContent('サーバーでエラーが発生しました')
    await user.click(screen.getByRole('button', { name: 'もう一度読み込む' }))
    expect(await screen.findByText('審査待ち 1件 · 古い順')).toBeInTheDocument()
  })
})
