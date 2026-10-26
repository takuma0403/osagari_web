import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderRoute } from '../../../test/renderRoute'
import { json } from '../../items/test/fixtures'
import type { TransactionListEntry } from '../api'
import { stubDealsFetch, transactionListEntry } from '../test/fixtures'
import { DealListPage } from './DealListPage'

const list =
  (entries: TransactionListEntry[], nextCursor: string | null = null) =>
  () =>
    json({ entries, nextCursor })

const buyerActive = [
  transactionListEntry({
    transactionId: 'tx1',
    status: 'in_deal',
    approvalDeadline: null,
    item: { shareId: 'a1', title: '線形代数入門', category: 'textbook', thumbnail: null },
  }),
  transactionListEntry({
    transactionId: 'tx2',
    item: { shareId: 'a2', title: '微分積分学 第3版', category: 'textbook', thumbnail: null },
  }),
]

const buyerEnded = [
  transactionListEntry({
    transactionId: 'tx3',
    status: 'expired',
    approvalDeadline: null,
    item: { shareId: 'a3', title: 'ミクロ経済学入門', category: 'textbook', thumbnail: null },
    price: 900,
  }),
]

const sellerActive = [
  transactionListEntry({
    transactionId: null,
    itemId: 'i1',
    requestCount: 3,
    approvalDeadline: null,
    item: { shareId: 'b1', title: '解剖学講義 改訂3版', category: 'textbook', thumbnail: null },
    price: 3500,
  }),
]

const buyer = {
  'GET /api/transactions?role=buyer&phase=active': list(buyerActive),
  'GET /api/transactions?role=buyer&phase=ended': list(buyerEnded),
}

const seller = {
  'GET /api/transactions?role=seller&phase=active': list(sellerActive),
  'GET /api/transactions?role=seller&phase=ended': list([]),
}

function setup(at = '/deals') {
  const view = renderRoute('/deals', <DealListPage />, {
    at,
    destinations: ['/deals/:dealId', '/sell/:itemId/requests', '/'],
  })
  return { ...view, user: userEvent.setup() }
}

const section = (name: string) => screen.getByRole('region', { name })

afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe('DealListPage', () => {
  it('購入タブで進行中と終了した取引を状態・補足つきで出し、行から取引詳細へ進む', async () => {
    vi.useFakeTimers({ now: Date.parse('2026-10-08T03:40:00Z'), shouldAdvanceTime: true })
    stubDealsFetch(buyer)
    const { user, router } = setup()
    expect(screen.getByRole('heading', { level: 1, name: '取引' })).toBeInTheDocument()
    expect(screen.getByRole('status')).toHaveTextContent('読み込み中…')
    expect(await screen.findByRole('heading', { name: '進行中' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: '購入' })).toHaveAttribute('aria-selected', 'true')

    const active = section('進行中')
    expect(within(active).getByText('取引中')).toBeInTheDocument()
    expect(within(active).getByText('受け渡しの日時をTeamsで相談')).toBeInTheDocument()
    expect(within(active).getByText('承認待ち')).toBeInTheDocument()
    expect(within(active).getByText('あと24時間で期限切れ')).toBeInTheDocument()
    const ended = section('終了した取引')
    expect(within(ended).getByText('期限切れ')).toBeInTheDocument()
    expect(within(ended).getByText('¥900')).toBeInTheDocument()

    await user.click(screen.getByRole('link', { name: /線形代数入門/ }))
    expect(router.state.location.pathname).toBe('/deals/tx1')
  })

  it('出品タブに切り替えると出品側を取り直し、商品ごとの購入希望の行から購入希望の一覧へ進む', async () => {
    const fetchMock = stubDealsFetch({ ...buyer, ...seller })
    const { user, router } = setup()
    await screen.findByRole('heading', { name: '進行中' })

    await user.click(screen.getByRole('tab', { name: '出品' }))
    expect(await screen.findByText('購入希望 3件')).toBeInTheDocument()
    expect(router.state.location.search).toBe('?tab=sell')
    expect(screen.getByText('相手を選んでください')).toBeInTheDocument()
    // 終了した取引がなければ、その見出しは出さない
    expect(screen.queryByRole('heading', { name: '終了した取引' })).not.toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/transactions?role=seller&phase=active',
      expect.anything(),
    )

    await user.click(screen.getByRole('link', { name: /解剖学講義/ }))
    expect(router.state.location.pathname).toBe('/sell/i1/requests')
  })

  it('?tab=sell で開くと出品タブを出す', async () => {
    stubDealsFetch(seller)
    setup('/deals?tab=sell')
    expect(await screen.findByText('購入希望 3件')).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: '出品' })).toHaveAttribute('aria-selected', 'true')
  })

  it.each([
    ['buyer', '/deals', '購入希望を出した商品はまだありません。'],
    ['seller', '/deals?tab=sell', '出品した商品の取引はまだありません。'],
  ])('%s の取引がなければ空の表示を出し、商品一覧へ案内する', async (role, at, message) => {
    stubDealsFetch({
      [`GET /api/transactions?role=${role}&phase=active`]: list([]),
      [`GET /api/transactions?role=${role}&phase=ended`]: list([]),
    })
    const { user, router } = setup(at)
    expect(await screen.findByText(message)).toBeInTheDocument()
    expect(screen.queryByRole('heading', { level: 2 })).not.toBeInTheDocument()
    await user.click(screen.getByRole('link', { name: '商品一覧を見る' }))
    expect(router.state.location.pathname).toBe('/')
  })

  it('読み込めなかったらエラーを出し、もう一度読み込める', async () => {
    let failed = false
    stubDealsFetch({
      ...buyer,
      'GET /api/transactions?role=buyer&phase=active': () => {
        if (failed) return list(buyerActive)()
        failed = true
        return json({ code: 'internal', message: 'サーバーでエラーが発生しました' }, 500)
      },
    })
    const { user } = setup()
    expect(await screen.findByRole('alert')).toHaveTextContent('サーバーでエラーが発生しました')
    await user.click(screen.getByRole('button', { name: 'もう一度読み込む' }))
    expect(await screen.findByRole('heading', { name: '進行中' })).toBeInTheDocument()
  })

  it('終了した取引の続きを「もっと見る」で cursor を渡して足す', async () => {
    stubDealsFetch({
      'GET /api/transactions?role=buyer&phase=active': list([]),
      'GET /api/transactions?role=buyer&phase=ended': list(buyerEnded, 'c1'),
      'GET /api/transactions?role=buyer&phase=ended&cursor=c1': list([
        transactionListEntry({
          transactionId: 'tx4',
          status: 'cancelled',
          approvalDeadline: null,
          item: { shareId: 'a4', title: '有機化学 演習', category: 'textbook', thumbnail: null },
        }),
      ]),
    })
    const { user } = setup()
    await screen.findByRole('heading', { name: '終了した取引' })
    expect(screen.queryByRole('heading', { name: '進行中' })).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'もっと見る' }))
    expect(await screen.findByText('有機化学 演習')).toBeInTheDocument()
    expect(screen.getByText('双方の同意で取り消し')).toBeInTheDocument()
    expect(screen.getByText('ミクロ経済学入門')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'もっと見る' })).not.toBeInTheDocument()
  })
})
