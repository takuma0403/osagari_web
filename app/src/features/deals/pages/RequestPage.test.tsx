import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderRoute } from '../../../test/renderRoute'
import { json, publicItem } from '../../items/test/fixtures'
import { bodyOf, stubDealsFetch, transaction } from '../test/fixtures'
import { RequestPage } from './RequestPage'

function setup() {
  const view = renderRoute('/items/:shareId/request', <RequestPage />, {
    at: '/items/abc123/request',
    destinations: ['/items/:shareId'],
  })
  return { ...view, user: userEvent.setup() }
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('RequestPage', () => {
  it('商品・支払い方法・お支払い予定額を出す', async () => {
    stubDealsFetch({ 'GET /api/items/shared/abc123': () => json(publicItem()) })
    setup()
    expect(await screen.findByText('線形代数入門')).toBeInTheDocument()
    expect(screen.getByText('決済はモック')).toBeInTheDocument()
    expect(screen.getByText('本庄キャンパスで受け渡し')).toBeInTheDocument()
    expect(screen.getByRole('radio', { name: /クレジットカード/ })).toBeChecked()
    expect(screen.getByRole('radio', { name: 'PayPay' })).not.toBeChecked()
    expect(screen.getByText('お支払い予定額')).toBeInTheDocument()
    expect(screen.getAllByText('¥1,500')).toHaveLength(2)
    expect(
      screen.getByText(
        '代金は、商品を受け取る際、QRコードを読み取った時点で出品者に支払われます。',
      ),
    ).toBeInTheDocument()
  })

  it('選んだ支払い方法で購入希望を送り、商品詳細へ戻ってトーストを出させる', async () => {
    const fetchMock = stubDealsFetch({
      'GET /api/items/shared/abc123': () => json(publicItem()),
      'POST /api/items/shared/abc123/requests': () => json(transaction(), 201),
    })
    const { user, router } = setup()
    await user.click(await screen.findByRole('radio', { name: 'PayPay' }))
    await user.click(screen.getByRole('button', { name: '購入希望を送る' }))
    expect(await screen.findByRole('heading', { name: '/items/abc123' })).toHaveAttribute(
      'data-state',
      JSON.stringify({ requestSent: true }),
    )
    expect(bodyOf(fetchMock, 'POST /api/items/shared/abc123/requests')).toEqual({
      paymentMethod: 'paypay',
    })
    // 戻るで購入希望の入力に戻らないよう、履歴を置き換える
    expect(router.state.historyAction).toBe('REPLACE')
  })

  it('送れなかったときはサーバーの文言を出し、画面に残る', async () => {
    stubDealsFetch({
      'GET /api/items/shared/abc123': () => json(publicItem()),
      'POST /api/items/shared/abc123/requests': () =>
        json(
          {
            code: 'invalid_status_transition',
            message: 'この商品にはすでに購入希望を出しています',
          },
          409,
        ),
    })
    const { user } = setup()
    await user.click(await screen.findByRole('button', { name: '購入希望を送る' }))
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'この商品にはすでに購入希望を出しています',
    )
    expect(screen.getByRole('button', { name: '購入希望を送る' })).toBeEnabled()
  })

  it.each([
    ['自分の出品', publicItem({ isOwn: true, itemId: 'i1' }), '自分の出品には購入希望を出せません'],
    ['取引中', publicItem({ status: 'in_deal' }), 'この商品は売り切れです'],
    ['取引完了', publicItem({ status: 'completed' }), 'この商品は売り切れです'],
  ])('%sの商品には送れない', async (_, item, reason) => {
    stubDealsFetch({ 'GET /api/items/shared/abc123': () => json(item) })
    setup()
    expect(await screen.findByText(reason)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '購入希望を送る' })).toBeDisabled()
  })

  it('見られない商品は「見つかりません」を出す', async () => {
    stubDealsFetch({
      'GET /api/items/shared/abc123': () =>
        json({ code: 'not_found', message: '見つかりません' }, 404),
    })
    setup()
    expect(await screen.findByRole('heading', { name: '商品が見つかりません' })).toBeInTheDocument()
  })
})
