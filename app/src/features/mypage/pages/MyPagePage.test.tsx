import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { CurrentUser } from '../../auth/api'
import type { TransactionListEntry } from '../../deals/api'
import { stubDealsFetch, transactionListEntry } from '../../deals/test/fixtures'
import { json } from '../../items/test/fixtures'
import { renderRoute } from '../../../test/renderRoute'
import type { OwnItemSummary } from '../api'
import { MyPagePage } from './MyPagePage'

const me: CurrentUser = {
  id: 'u1',
  email: '26238001@edu.cc.saga-u.ac.jp',
  studentNumber: '26238001',
  admissionYear: 2026,
  departmentCode: '238',
  department: '238',
  departmentName: '理工学部 理工学科',
  expiresOn: '2030-03-31',
  status: 'active',
  sellerStatus: 'pending',
}

const ownItems: OwnItemSummary[] = [
  {
    id: 'i1',
    shareId: 'a1',
    status: 'requested',
    title: '線形代数入門',
    price: 1500,
    thumbnail: null,
    requestCount: 3,
    updatedAt: '2026-10-07T03:40:00Z',
  },
  {
    id: 'i2',
    shareId: 'a2',
    status: 'draft',
    title: '電気スタンド',
    price: null,
    thumbnail: null,
    requestCount: 0,
    updatedAt: '2026-10-06T03:40:00Z',
  },
]

const entry = (id: string, status: TransactionListEntry['status'], title: string) =>
  transactionListEntry({
    transactionId: id,
    status,
    approvalDeadline: null,
    item: { shareId: `s-${id}`, title, category: 'textbook', thumbnail: null },
  })

const activePath = 'GET /api/transactions?role=buyer&phase=active'
const endedPath = 'GET /api/transactions?role=buyer&phase=ended&limit=50'

const handlers = {
  'GET /api/me/items': () => json({ items: ownItems }),
  [activePath]: () =>
    json({
      entries: [entry('tx1', 'in_deal', '微分積分学'), entry('tx2', 'requested', '物理学')],
      nextCursor: null,
    }),
  [endedPath]: () =>
    json({
      entries: [entry('tx3', 'expired', '化学'), entry('tx4', 'completed', 'ローテーブル')],
      nextCursor: 'c2',
    }),
  'GET /api/transactions?role=buyer&phase=ended&cursor=c2&limit=50': () =>
    json({ entries: [entry('tx5', 'completed', '電子レンジ')], nextCursor: null }),
}

function setup(at = '/mypage', overrides: Partial<CurrentUser> = {}) {
  return renderRoute('/mypage', <MyPagePage />, {
    at,
    session: { user: { ...me, ...overrides } },
  })
}

describe('MyPagePage', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('学籍番号・学部・入学年度とアカウントの有効期限を出す', () => {
    stubDealsFetch(handlers)
    setup()
    const profile = screen.getByRole('region', { name: 'プロフィール' })
    expect(within(profile).getByText('26238001')).toBeInTheDocument()
    expect(within(profile).getByText('理工学部 理工学科 · 2026年度入学')).toBeInTheDocument()
    expect(within(profile).getByText('2030年3月31日')).toBeInTheDocument()
  })

  it('出品した商品を状態のバッジと行き先つきで出す', async () => {
    stubDealsFetch(handlers)
    setup()
    expect(await screen.findByRole('tab', { name: '出品した商品 2' })).toHaveAttribute(
      'aria-selected',
      'true',
    )
    const requested = screen.getByRole('link', { name: /線形代数入門/ })
    expect(requested).toHaveAttribute('href', '/sell/i1/requests')
    expect(within(requested).getByText('購入希望 3件')).toBeInTheDocument()
    const draft = screen.getByRole('link', { name: /電気スタンド/ })
    expect(draft).toHaveAttribute('href', '/sell/i2')
    expect(within(draft).getByText('価格未設定')).toBeInTheDocument()
    expect(within(draft).getByText('下書き')).toBeInTheDocument()
  })

  it('購入した商品は終了した取引を最後まで読み、取引中・完了だけを出す', async () => {
    stubDealsFetch(handlers)
    setup()
    await userEvent.click(await screen.findByRole('tab', { name: '購入した商品 3' }))
    const panel = screen.getByRole('tabpanel')
    const links = within(panel).getAllByRole('link')
    expect(links.map((l) => l.getAttribute('href'))).toEqual([
      '/deals/tx1',
      '/deals/tx4',
      '/deals/tx5',
    ])
    expect(within(panel).queryByText('物理学')).not.toBeInTheDocument()
    expect(within(panel).queryByText('化学')).not.toBeInTheDocument()
  })

  it('?tab=bought なら購入した商品を選んで開く', async () => {
    stubDealsFetch(handlers)
    setup('/mypage?tab=bought')
    expect(await screen.findByRole('tab', { name: '購入した商品 3' })).toHaveAttribute(
      'aria-selected',
      'true',
    )
  })

  it('履歴がなければ空の案内を出す', async () => {
    stubDealsFetch({
      'GET /api/me/items': () => json({ items: [] }),
      [activePath]: () => json({ entries: [], nextCursor: null }),
      [endedPath]: () => json({ entries: [entry('tx3', 'cancelled', '化学')], nextCursor: null }),
    })
    setup()
    expect(await screen.findByText('出品した商品はまだありません。')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: '出品する' })).toHaveAttribute('href', '/sell')
    await userEvent.click(screen.getByRole('tab', { name: '購入した商品 0' }))
    expect(screen.getByText('購入した商品はまだありません。')).toBeInTheDocument()
  })

  it('読み込みに失敗したら案内を出し、もう一度読み込める', async () => {
    let failed = false
    stubDealsFetch({
      ...handlers,
      'GET /api/me/items': () => {
        if (failed) return json({ items: ownItems })
        failed = true
        return json({ code: 'internal', message: 'サーバーでエラーが起きました' }, 500)
      },
    })
    setup()
    expect(await screen.findByRole('alert')).toHaveTextContent('サーバーでエラーが起きました')
    await userEvent.click(screen.getByRole('button', { name: 'もう一度読み込む' }))
    expect(await screen.findByRole('tab', { name: '出品した商品 2' })).toBeInTheDocument()
  })

  it('売上管理と学生証の登録へ進める。学生証は審査状況を添える', () => {
    stubDealsFetch(handlers)
    setup()
    const nav = screen.getByRole('navigation', { name: '設定' })
    expect(within(nav).getByRole('link', { name: /売上管理/ })).toHaveAttribute(
      'href',
      '/mypage/sales',
    )
    const card = within(nav).getByRole('link', { name: /学生証の登録/ })
    expect(card).toHaveAttribute('href', '/mypage/student-card')
    expect(card).toHaveTextContent('審査中')
  })
})
