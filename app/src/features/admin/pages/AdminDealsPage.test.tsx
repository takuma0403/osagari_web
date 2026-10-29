import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { AdminLayout } from '../../../layouts/AdminLayout/AdminLayout'
import type { AdminItem, AdminTransaction } from '../api'
import { AdminDealsPage } from './AdminDealsPage'

const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })

const itemOf = (id: string, title: string, overrides: Partial<AdminItem> = {}) =>
  ({
    id,
    shareId: `s-${id}`,
    title,
    price: 6000,
    status: 'published',
    sellerStudentNumber: '25238117',
    thumbnail: null,
    createdAt: '2026-10-07T03:00:00Z',
    ...overrides,
  }) satisfies AdminItem

const txOf = (id: string, title: string, overrides: Partial<AdminTransaction> = {}) =>
  ({
    id,
    status: 'in_deal',
    item: { shareId: `s-${id}`, title, status: 'in_deal' },
    sellerStudentNumber: '25238117',
    buyerStudentNumber: '26141023',
    price: 1500,
    paymentMethod: 'card',
    feeRateBasisPoints: 500,
    buyerCancelAgreedAt: '2026-10-08T05:02:00Z',
    sellerCancelAgreedAt: null,
    completedAt: null,
    createdAt: '2026-10-03T00:00:00Z',
    updatedAt: '2026-10-08T05:02:00Z',
    ...overrides,
  }) satisfies AdminTransaction

const fridge = itemOf('i1', '一人暮らし用 冷蔵庫 2ドア 138L')
const soldOut = itemOf('i2', '携帯ゲーム機 本体＋ソフト2本', { status: 'completed', price: 15000 })
const linear = txOf('t1', '線形代数入門')
const done = txOf('t2', '有機化学 演習', {
  status: 'completed',
  buyerCancelAgreedAt: null,
  completedAt: '2026-09-28T03:40:00Z',
})

type Handler = (method: string, url: string) => Response | undefined

function setup(handler: Handler = () => undefined) {
  const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
    const method = init?.method ?? 'GET'
    const res = handler(method, url)
    if (res) return res
    if (url === '/api/admin/items') return json({ entries: [fridge, soldOut], nextCursor: null })
    if (url === '/api/admin/transactions')
      return json({ entries: [linear, done], nextCursor: null })
    throw new Error(`想定していない呼び出し: ${method} ${url}`)
  })
  vi.stubGlobal('fetch', fetchMock)
  const router = createMemoryRouter(
    [
      {
        element: <AdminLayout />,
        children: [{ path: '/admin/items', element: <AdminDealsPage /> }],
      },
    ],
    { initialEntries: ['/admin/items'] },
  )
  render(<RouterProvider router={router} />)
  return { fetchMock, user: userEvent.setup() }
}

const calls = (fetchMock: ReturnType<typeof setup>['fetchMock'], method: string, url: string) =>
  fetchMock.mock.calls.filter(([u, init]) => u === url && (init?.method ?? 'GET') === method)

describe('AdminDealsPage', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('最初は出品のタブで、出品を表に出して先頭の詳細を出す', async () => {
    setup()
    expect(screen.getByRole('tab', { name: '出品' })).toHaveAttribute('aria-selected', 'true')
    const table = await screen.findByRole('region', { name: '出品一覧' })
    const rows = within(table).getAllByRole('row')
    expect(rows[1]).toHaveTextContent('一人暮らし用 冷蔵庫 2ドア 138L25238117¥6,000公開中')
    expect(rows[2]).toHaveTextContent('取引完了')
    const panel = screen.getByRole('region', { name: '出品の詳細' })
    expect(
      within(panel).getByRole('heading', { name: '一人暮らし用 冷蔵庫 2ドア 138L' }),
    ).toBeInTheDocument()
    expect(within(panel).getByText('写真なし')).toBeInTheDocument()
  })

  it('出品の停止は確認してから行う', async () => {
    const { fetchMock, user } = setup((method, url) => {
      if (method === 'POST' && url === '/api/admin/items/i1/suspend')
        return json({ ...fridge, status: 'suspended' })
    })
    await user.click(await screen.findByRole('button', { name: '出品を停止' }))
    const dialog = screen.getByRole('dialog', { name: '出品を停止しますか？' })
    expect(calls(fetchMock, 'POST', '/api/admin/items/i1/suspend')).toHaveLength(0)
    await user.click(within(dialog).getByRole('button', { name: '停止する' }))
    expect(
      await screen.findByText('「一人暮らし用 冷蔵庫 2ドア 138L」を停止しました'),
    ).toBeInTheDocument()
    expect(within(screen.getAllByRole('row')[1]).getByText('停止')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '出品を停止' })).toBeDisabled()
  })

  it('取引完了の出品は停止できない', async () => {
    const { user } = setup()
    await user.click(await screen.findByRole('button', { name: '携帯ゲーム機 本体＋ソフト2本' }))
    expect(screen.getByRole('button', { name: '出品を停止' })).toBeDisabled()
  })

  it('取引のタブで、取引とキャンセル申請の状況を出す', async () => {
    const { user } = setup()
    await screen.findByRole('region', { name: '出品一覧' })
    await user.click(screen.getByRole('tab', { name: '取引' }))
    const table = await screen.findByRole('region', { name: '取引一覧' })
    expect(within(table).getAllByRole('row')[1]).toHaveTextContent(
      '線形代数入門25238117 → 26141023¥1,500キャンセル申請中',
    )
    const panel = screen.getByRole('region', { name: '取引の詳細' })
    expect(within(panel).getByText('決済はモック')).toBeInTheDocument()
    expect(within(panel).getByText('5%（この取引に適用）')).toBeInTheDocument()
    expect(within(panel).getByText('購入者が申請（出品者は未同意）')).toBeInTheDocument()
  })

  it('取引のキャンセル・返金は確認してから行う', async () => {
    const { fetchMock, user } = setup((method, url) => {
      if (method === 'POST' && url === '/api/admin/transactions/t1/cancel')
        return json({ ...linear, status: 'cancelled' })
    })
    await screen.findByRole('region', { name: '出品一覧' })
    await user.click(screen.getByRole('tab', { name: '取引' }))
    await user.click(
      await screen.findByRole('button', { name: 'キャンセル・返金（モック）を行う' }),
    )
    const dialog = screen.getByRole('dialog', { name: '取引をキャンセルしますか？' })
    await user.click(within(dialog).getByRole('button', { name: 'キャンセル・返金する' }))
    expect(
      await screen.findByText('「線形代数入門」の取引をキャンセルし、返金しました'),
    ).toBeInTheDocument()
    expect(calls(fetchMock, 'POST', '/api/admin/transactions/t1/cancel')).toHaveLength(1)
    expect(
      within(screen.getByRole('region', { name: '取引の詳細' })).getByText('キャンセル'),
    ).toBeInTheDocument()
  })

  it('受け渡し完了後の取引はキャンセルできない', async () => {
    const { user } = setup()
    await screen.findByRole('region', { name: '出品一覧' })
    await user.click(screen.getByRole('tab', { name: '取引' }))
    await user.click(await screen.findByRole('button', { name: '有機化学 演習' }))
    expect(screen.getByRole('button', { name: 'キャンセル・返金（モック）を行う' })).toBeDisabled()
  })

  it('状態が変わっていたら一覧を取り直し、理由を出す', async () => {
    const { fetchMock, user } = setup((method, url) => {
      if (method === 'POST' && url === '/api/admin/transactions/t1/cancel')
        return json(
          { code: 'invalid_status_transition', message: '取引の状態が変わっています' },
          409,
        )
    })
    await screen.findByRole('region', { name: '出品一覧' })
    await user.click(screen.getByRole('tab', { name: '取引' }))
    await user.click(
      await screen.findByRole('button', { name: 'キャンセル・返金（モック）を行う' }),
    )
    await user.click(
      within(screen.getByRole('dialog')).getByRole('button', { name: 'キャンセル・返金する' }),
    )
    expect(await screen.findByText('取引の状態が変わっています')).toBeInTheDocument()
    await screen.findByRole('region', { name: '取引一覧' })
    expect(calls(fetchMock, 'GET', '/api/admin/transactions')).toHaveLength(2)
  })

  it('出品がなければそう出す', async () => {
    setup((_method, url) => {
      if (url === '/api/admin/items') return json({ entries: [], nextCursor: null })
    })
    expect(await screen.findByText('出品はまだありません。')).toBeInTheDocument()
  })
})
