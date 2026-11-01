import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderRoute } from '../../../test/renderRoute'
import { json } from '../../items/test/fixtures'
import {
  approvedTransaction,
  ownItem,
  purchaseRequest,
  stubDealsFetch,
  transaction,
} from '../test/fixtures'
import { DealRequestsPage } from './DealRequestsPage'

const requests = [
  purchaseRequest(),
  purchaseRequest({
    transactionId: 'tx2',
    buyer: { departmentName: '理工学部', admissionYear: 2025, completedDealCount: 2 },
  }),
]

const loaded = {
  'GET /api/items/i1': () => json(ownItem()),
  'GET /api/items/i1/requests': () => json({ requests }),
}

function setup() {
  const view = renderRoute('/sell/:itemId/requests', <DealRequestsPage />, {
    at: '/sell/i1/requests',
    destinations: ['/deals/:dealId', '/items/:shareId'],
  })
  return { ...view, user: userEvent.setup() }
}

const card = (name: string) => screen.getByRole('region', { name: `${name}の購入希望` })

afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe('DealRequestsPage', () => {
  it('商品と購入希望を出し、購入者の学籍番号は出さない', async () => {
    vi.useFakeTimers({ now: Date.parse('2026-10-08T03:40:00Z'), shouldAdvanceTime: true })
    stubDealsFetch(loaded)
    setup()
    expect(await screen.findByRole('heading', { name: '購入希望 2件' })).toBeInTheDocument()
    expect(screen.getByText('線形代数入門')).toBeInTheDocument()
    expect(screen.getByText('あなたが出品')).toBeInTheDocument()
    expect(screen.getByText('購入希望あり').closest('li')).toHaveAttribute('aria-current', 'step')
    expect(screen.getByText('承認できるのは1人だけです')).toBeInTheDocument()
    const first = card('経済学部')
    expect(within(first).getByText('経済学部 · 2026年度入学')).toBeInTheDocument()
    expect(within(first).getByText(/に希望 · 取引完了 0回/)).toBeInTheDocument()
    expect(within(first).getByText('期限まで残り 24時間')).toBeInTheDocument()
    expect(within(card('理工学部')).getByText(/取引完了 2回/)).toBeInTheDocument()
    expect(document.body).not.toHaveTextContent(/\d{8}/)
  })

  it('確認してから承認し、確認モーダルに購入者の学籍番号を出す。「確認しました」で記録して取引詳細へ進む', async () => {
    const fetchMock = stubDealsFetch({
      ...loaded,
      'POST /api/transactions/tx2/approve': () => json(approvedTransaction({ id: 'tx2' })),
      'POST /api/transactions/tx2/acknowledge': () =>
        json(approvedTransaction({ id: 'tx2', sellerAcknowledgedAt: '2026-10-08T03:41:00Z' })),
    })
    const { user, router } = setup()
    await screen.findByRole('heading', { name: '購入希望 2件' })
    await user.click(within(card('理工学部')).getByRole('button', { name: 'この人に決める' }))

    const confirm = screen.getByRole('dialog', { name: 'この人に決めますか？' })
    expect(fetchMock).not.toHaveBeenCalledWith('/api/transactions/tx2/approve', expect.anything())
    await user.click(within(confirm).getByRole('button', { name: 'この人に決める' }))

    const sheet = await screen.findByRole('dialog', { name: '取引が成立しました' })
    expect(within(sheet).getByText('26141023')).toBeInTheDocument()
    expect(within(sheet).getByText(/代金 ¥1,500 は運営が預かっています。/)).toBeInTheDocument()
    expect(
      within(sheet).getByText(
        '入金は、受け渡しの場で購入者がQRコードを読み取った時点で確定します。受け渡しの際は、忘れずにQRコードを読み取ってもらってください。',
      ),
    ).toBeInTheDocument()

    // 確認しましたを押すまで閉じない
    await user.keyboard('{Escape}')
    expect(screen.getByRole('dialog', { name: '取引が成立しました' })).toBeInTheDocument()

    await user.click(within(sheet).getByRole('button', { name: '確認しました' }))
    expect(await screen.findByRole('heading', { name: '/deals/tx2' })).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/transactions/tx2/acknowledge',
      expect.objectContaining({ method: 'POST' }),
    )
    expect(router.state.historyAction).toBe('REPLACE')
  })

  it('「確認しました」を記録できなければ文言を出し、モーダルに残る', async () => {
    stubDealsFetch({
      ...loaded,
      'POST /api/transactions/tx1/approve': () => json(approvedTransaction()),
      'POST /api/transactions/tx1/acknowledge': () =>
        json({ code: 'internal_error', message: 'サーバーでエラーが発生しました' }, 500),
    })
    const { user, router } = setup()
    await screen.findByRole('heading', { name: '購入希望 2件' })
    await user.click(within(card('経済学部')).getByRole('button', { name: 'この人に決める' }))
    await user.click(
      within(screen.getByRole('dialog', { name: 'この人に決めますか？' })).getByRole('button', {
        name: 'この人に決める',
      }),
    )
    const sheet = await screen.findByRole('dialog', { name: '取引が成立しました' })
    await user.click(within(sheet).getByRole('button', { name: '確認しました' }))
    expect(await within(sheet).findByRole('alert')).toHaveTextContent(
      'サーバーでエラーが発生しました',
    )
    expect(within(sheet).getByRole('button', { name: '確認しました' })).toBeEnabled()
    expect(router.state.location.pathname).toBe('/sell/i1/requests')
  })

  it('学籍番号をコピーできる', async () => {
    stubDealsFetch({
      ...loaded,
      'POST /api/transactions/tx1/approve': () => json(approvedTransaction()),
    })
    const { user } = setup()
    await screen.findByRole('heading', { name: '購入希望 2件' })
    await user.click(within(card('経済学部')).getByRole('button', { name: 'この人に決める' }))
    await user.click(
      within(screen.getByRole('dialog', { name: 'この人に決めますか？' })).getByRole('button', {
        name: 'この人に決める',
      }),
    )
    const sheet = await screen.findByRole('dialog', { name: '取引が成立しました' })
    await user.click(within(sheet).getByRole('button', { name: 'コピー' }))
    expect(await within(sheet).findByRole('button', { name: 'コピーしました' })).toBeInTheDocument()
    expect(await navigator.clipboard.readText()).toBe('26141023')
  })

  it('承認の確認でやめたら承認しない', async () => {
    const fetchMock = stubDealsFetch(loaded)
    const { user } = setup()
    await screen.findByRole('heading', { name: '購入希望 2件' })
    await user.click(within(card('経済学部')).getByRole('button', { name: 'この人に決める' }))
    await user.click(screen.getByRole('button', { name: 'やめる' }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledTimes(2)
  })

  it('見送ると確認してから却下し、一覧を取り直す', async () => {
    let remaining = requests
    const fetchMock = stubDealsFetch({
      'GET /api/items/i1': () => json(ownItem()),
      'GET /api/items/i1/requests': () => json({ requests: remaining }),
      'POST /api/transactions/tx1/reject': () => {
        remaining = requests.slice(1)
        return json(transaction({ role: 'seller', status: 'rejected' }))
      },
    })
    const { user } = setup()
    await screen.findByRole('heading', { name: '購入希望 2件' })
    await user.click(within(card('経済学部')).getByRole('button', { name: '見送る' }))
    const confirm = screen.getByRole('dialog', { name: 'この購入希望を見送りますか？' })
    await user.click(within(confirm).getByRole('button', { name: '見送る' }))
    expect(await screen.findByRole('heading', { name: '購入希望 1件' })).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/transactions/tx1/reject',
      expect.objectContaining({ method: 'POST' }),
    )
    expect(screen.queryByRole('region', { name: '経済学部の購入希望' })).not.toBeInTheDocument()
  })

  it('状態が変わっていて承認できなかったら文言を出して一覧を取り直す', async () => {
    let remaining = requests
    const fetchMock = stubDealsFetch({
      'GET /api/items/i1': () => json(ownItem()),
      'GET /api/items/i1/requests': () => json({ requests: remaining }),
      'POST /api/transactions/tx1/approve': () => {
        remaining = requests.slice(1)
        return json({ code: 'invalid_status_transition', message: '承認期限を過ぎています' }, 409)
      },
    })
    const { user } = setup()
    await screen.findByRole('heading', { name: '購入希望 2件' })
    await user.click(within(card('経済学部')).getByRole('button', { name: 'この人に決める' }))
    await user.click(
      within(screen.getByRole('dialog', { name: 'この人に決めますか？' })).getByRole('button', {
        name: 'この人に決める',
      }),
    )
    expect(await screen.findByRole('alert')).toHaveTextContent('承認期限を過ぎています')
    expect(await screen.findByRole('heading', { name: '購入希望 1件' })).toBeInTheDocument()
    expect(fetchMock.mock.calls.filter(([url]) => url === '/api/items/i1/requests')).toHaveLength(2)
  })

  it('購入希望がなければそう出す', async () => {
    stubDealsFetch({
      'GET /api/items/i1': () => json(ownItem({ status: 'published' })),
      'GET /api/items/i1/requests': () => json({ requests: [] }),
    })
    setup()
    expect(await screen.findByText('承認を待っている購入希望はありません。')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: '商品ページを見る' })).toHaveAttribute(
      'href',
      '/items/abc123',
    )
  })

  it('ほかの人の商品は「見つかりません」を出す', async () => {
    stubDealsFetch({
      'GET /api/items/i1': () => json({ code: 'not_found', message: '見つかりません' }, 404),
      'GET /api/items/i1/requests': () =>
        json({ code: 'not_found', message: '見つかりません' }, 404),
    })
    setup()
    expect(await screen.findByRole('heading', { name: '商品が見つかりません' })).toBeInTheDocument()
  })
})
