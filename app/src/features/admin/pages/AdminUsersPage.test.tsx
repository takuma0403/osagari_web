import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { AdminLayout } from '../../../layouts/AdminLayout/AdminLayout'
import type { AdminUser, AdminUserList } from '../api'
import { AdminUsersPage } from './AdminUsersPage'

const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })

const userOf = (id: string, studentNumber: string, overrides: Partial<AdminUser> = {}) =>
  ({
    id,
    studentNumber,
    departmentName: '理工学部 理工学科',
    admissionYear: 2025,
    expiresOn: '2029-03-31',
    status: 'active',
    sellerStatus: 'approved',
    cancelCount: 4,
    activeTransactionCount: 1,
    withdrawalRequestedAt: null,
    createdAt: '2026-04-01T00:00:00Z',
    ...overrides,
  }) satisfies AdminUser

const first = userOf('u1', '25238117', { withdrawalRequestedAt: '2026-10-01T00:00:00Z' })
const second = userOf('u2', '24101032', {
  status: 'suspended',
  departmentName: '教育学部 学校教育課程',
})
const listOf = (entries: AdminUser[], nextCursor: string | null = null): AdminUserList => ({
  entries,
  withdrawalRequestCount: 2,
  nextCursor,
})

type Handler = (method: string, url: string, init?: RequestInit) => Response | undefined

/** 一覧は既定で2人を返す。handler が返さなかった呼び出しは失敗させる */
function setup(handler: Handler = () => undefined) {
  const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
    const method = init?.method ?? 'GET'
    const res = handler(method, url, init)
    if (res) return res
    if (method === 'GET' && url.startsWith('/api/admin/users')) return json(listOf([first, second]))
    throw new Error(`想定していない呼び出し: ${method} ${url}`)
  })
  vi.stubGlobal('fetch', fetchMock)
  const router = createMemoryRouter(
    [
      {
        element: <AdminLayout />,
        children: [{ path: '/admin/users', element: <AdminUsersPage /> }],
      },
    ],
    { initialEntries: ['/admin/users'] },
  )
  render(<RouterProvider router={router} />)
  return { fetchMock, user: userEvent.setup() }
}

const calls = (fetchMock: ReturnType<typeof setup>['fetchMock'], method: string, url: string) =>
  fetchMock.mock.calls.filter(([u, init]) => u === url && (init?.method ?? 'GET') === method)

const panel = () => screen.getByRole('region', { name: 'ユーザーの詳細' })

describe('AdminUsersPage', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('ユーザーを表に出し、先頭を選んで詳細を出す', async () => {
    setup()
    expect(screen.getByText('読み込み中…')).toHaveAttribute('role', 'status')
    expect(await screen.findByText('退学申請 2件')).toBeInTheDocument()
    const table = screen.getByRole('region', { name: 'ユーザー一覧' })
    const rows = within(table).getAllByRole('row')
    expect(rows).toHaveLength(3)
    expect(rows[1]).toHaveTextContent('25238117理工学部有効承認済み2029年3月31日4')
    expect(within(rows[1]).getByRole('button', { name: '25238117' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
    expect(within(panel()).getByText('退学申請あり')).toBeInTheDocument()
    expect(within(panel()).getByText('理工学部 理工学科')).toBeInTheDocument()
    expect(within(panel()).getByText('1件')).toBeInTheDocument()
    expect(within(panel()).getByLabelText('有効期限の変更')).toHaveValue('2029-03-31')
  })

  it('検索は数字だけを受け付け、Enter で学籍番号の前方一致を送る', async () => {
    const { fetchMock, user } = setup()
    await screen.findByText('退学申請 2件')
    await user.type(screen.getByRole('searchbox'), '25a23{Enter}')
    expect(await screen.findByRole('region', { name: 'ユーザー一覧' })).toBeInTheDocument()
    expect(calls(fetchMock, 'GET', '/api/admin/users?q=2523')).toHaveLength(1)
  })

  it('状態で絞り込む', async () => {
    const { fetchMock, user } = setup()
    await screen.findByText('退学申請 2件')
    const filters = screen.getByRole('group', { name: '状態で絞り込み' })
    expect(within(filters).getByRole('button', { name: 'すべて' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
    await user.click(within(filters).getByRole('button', { name: '退学申請' }))
    await screen.findByRole('region', { name: 'ユーザー一覧' })
    expect(calls(fetchMock, 'GET', '/api/admin/users?filter=withdrawal_requested')).toHaveLength(1)
  })

  it('「もっと見る」で続きを足す', async () => {
    const third = userOf('u3', '22238090', { status: 'expired' })
    const { user } = setup((_method, url) => {
      if (url === '/api/admin/users') return json(listOf([first, second], 'c2'))
      if (url === '/api/admin/users?cursor=c2') return json(listOf([third]))
    })
    await user.click(await screen.findByRole('button', { name: 'もっと見る' }))
    expect(await screen.findByRole('button', { name: '22238090' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'もっと見る' })).not.toBeInTheDocument()
  })

  it('有効期限を変更する', async () => {
    const { fetchMock, user } = setup((method, url) => {
      if (method === 'PUT' && url === '/api/admin/users/u1/expires-on')
        return json({ ...first, expiresOn: '2030-03-31' })
    })
    const input = await screen.findByLabelText('有効期限の変更')
    const button = within(panel()).getByRole('button', { name: '変更する' })
    expect(button).toBeDisabled()
    await user.clear(input)
    await user.type(input, '2030-03-31')
    await user.click(button)
    expect(
      await screen.findByText('25238117 の有効期限を2030年3月31日に変更しました'),
    ).toBeInTheDocument()
    const [[, init]] = calls(fetchMock, 'PUT', '/api/admin/users/u1/expires-on')
    expect(JSON.parse(String(init?.body))).toEqual({ expiresOn: '2030-03-31' })
    expect(within(screen.getAllByRole('row')[1]).getByText('2030年3月31日')).toBeInTheDocument()
  })

  it('過去の日付などで失敗したら、サーバーの文を出す', async () => {
    const { user } = setup((method, url) => {
      if (method === 'PUT' && url === '/api/admin/users/u1/expires-on')
        return json({ code: 'bad_request', message: '今日以降の日付にしてください' }, 400)
    })
    const input = await screen.findByLabelText('有効期限の変更')
    await user.clear(input)
    await user.type(input, '2020-03-31')
    await user.click(within(panel()).getByRole('button', { name: '変更する' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('今日以降の日付にしてください')
  })

  it('停止は確認してから行い、停止中には「停止を解除」を出す', async () => {
    const { fetchMock, user } = setup((method, url) => {
      if (method === 'POST' && url === '/api/admin/users/u1/suspend')
        return json({ ...first, status: 'suspended' })
    })
    await user.click(await screen.findByRole('button', { name: 'アカウントを停止' }))
    const dialog = screen.getByRole('dialog', { name: 'アカウントを停止しますか？' })
    expect(calls(fetchMock, 'POST', '/api/admin/users/u1/suspend')).toHaveLength(0)
    await user.click(within(dialog).getByRole('button', { name: '停止する' }))
    expect(await screen.findByText('25238117 を停止しました')).toBeInTheDocument()
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(within(panel()).getByRole('button', { name: '停止を解除' })).toBeInTheDocument()
    expect(within(panel()).getByText('停止')).toBeInTheDocument()
  })

  it('停止の解除は確認せずに行う', async () => {
    const { user } = setup((method, url) => {
      if (method === 'POST' && url === '/api/admin/users/u2/unsuspend')
        return json({ ...second, status: 'active' })
    })
    await user.click(await screen.findByRole('button', { name: '24101032' }))
    await user.click(within(panel()).getByRole('button', { name: '停止を解除' }))
    expect(await screen.findByText('24101032 の停止を解除しました')).toBeInTheDocument()
    expect(within(panel()).getByRole('button', { name: 'アカウントを停止' })).toBeInTheDocument()
  })

  it('退学による削除は確認してから行い、申請の件数を減らして操作を出さない', async () => {
    const { user } = setup((method, url) => {
      if (method === 'POST' && url === '/api/admin/users/u1/delete')
        return json({ ...first, status: 'deleted' })
    })
    await user.click(await screen.findByRole('button', { name: '退学によりアカウントを削除' }))
    const dialog = screen.getByRole('dialog', { name: '退学によりアカウントを削除しますか？' })
    await user.click(within(dialog).getByRole('button', { name: '削除する' }))
    expect(await screen.findByText('25238117 を削除しました')).toBeInTheDocument()
    expect(screen.getByText('退学申請 1件')).toBeInTheDocument()
    expect(within(panel()).queryByText('退学申請あり')).not.toBeInTheDocument()
    expect(
      within(panel()).getByText('失効・削除したアカウントは操作できません。'),
    ).toBeInTheDocument()
  })

  it('状態が変わっていたら一覧を取り直し、理由を出す', async () => {
    const { fetchMock, user } = setup((method, url) => {
      if (method === 'POST' && url === '/api/admin/users/u1/suspend')
        return json({ code: 'invalid_status_transition', message: '状態が変わっています' }, 409)
    })
    await user.click(await screen.findByRole('button', { name: 'アカウントを停止' }))
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: '停止する' }))
    expect(await screen.findByText('状態が変わっています')).toBeInTheDocument()
    await screen.findByRole('region', { name: 'ユーザー一覧' })
    expect(calls(fetchMock, 'GET', '/api/admin/users')).toHaveLength(2)
  })

  it('読み込みに失敗したら、読み直せる', async () => {
    let failed = false
    const { user } = setup((_method, url) => {
      if (url === '/api/admin/users' && !failed) {
        failed = true
        return json({ code: 'internal', message: 'エラーが起きました' }, 500)
      }
    })
    expect(await screen.findByRole('alert')).toHaveTextContent('エラーが起きました')
    await user.click(screen.getByRole('button', { name: 'もう一度読み込む' }))
    expect(await screen.findByText('退学申請 2件')).toBeInTheDocument()
  })

  it('条件に合うユーザーがいなければそう出す', async () => {
    setup((_method, url) => {
      if (url === '/api/admin/users') return json(listOf([]))
    })
    expect(await screen.findByText('条件に合うユーザーはいません。')).toBeInTheDocument()
    expect(screen.queryByRole('region', { name: 'ユーザーの詳細' })).not.toBeInTheDocument()
  })
})
