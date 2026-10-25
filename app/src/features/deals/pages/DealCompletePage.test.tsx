import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderRoute } from '../../../test/renderRoute'
import { json } from '../../items/test/fixtures'
import { bodyOf, buyerDealTransaction, stubDealsFetch } from '../test/fixtures'
import { DealCompletePage } from './DealCompletePage'

const verified = () => new Response(null, { status: 204 })
const invalid = () => json({ code: 'invalid_handover_token', message: '使えません' }, 400)

function setup(at: string) {
  const view = renderRoute('/deals/:dealId/complete', <DealCompletePage />, {
    at,
    destinations: ['/deals/:dealId'],
  })
  return { ...view, user: userEvent.setup() }
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('DealCompletePage', () => {
  it('完了リンク（#t=）を開いたら照合して URL からトークンを消し、「受け取った」で完了する', async () => {
    const fetchMock = stubDealsFetch({
      'GET /api/transactions/tx1': () => json(buyerDealTransaction()),
      'POST /api/transactions/tx1/handover/verify': verified,
      'POST /api/transactions/tx1/complete': () =>
        json(buyerDealTransaction({ status: 'completed', handoverMethod: 'link' })),
    })
    const { user, router } = setup('/deals/tx1/complete#t=link-token')
    const dialog = await screen.findByRole('dialog', { name: '商品を受け取りましたか？' })
    expect(dialog).toHaveTextContent('¥1,500')
    expect(router.state.location.hash).toBe('')
    expect(screen.getByText('25238117')).toBeInTheDocument()
    expect(bodyOf(fetchMock, 'POST /api/transactions/tx1/handover/verify')).toEqual({
      kind: 'link',
      secret: 'link-token',
    })

    await user.click(screen.getByRole('button', { name: '受け取った' }))
    expect(bodyOf(fetchMock, 'POST /api/transactions/tx1/complete')).toEqual({
      kind: 'link',
      secret: 'link-token',
    })
    expect(router.state.location.pathname).toBe('/deals/tx1')
  })

  it('標準のカメラで読んだ受け渡しQR（#q=）は qr として照合する', async () => {
    const fetchMock = stubDealsFetch({
      'GET /api/transactions/tx1': () => json(buyerDealTransaction()),
      'POST /api/transactions/tx1/handover/verify': verified,
    })
    setup('/deals/tx1/complete#q=qr-token')
    await screen.findByRole('dialog', { name: '商品を受け取りましたか？' })
    expect(bodyOf(fetchMock, 'POST /api/transactions/tx1/handover/verify')).toEqual({
      kind: 'qr',
      secret: 'qr-token',
    })
  })

  it('使えない完了リンクは、発行し直してもらうよう案内する', async () => {
    stubDealsFetch({
      'GET /api/transactions/tx1': () => json(buyerDealTransaction()),
      'POST /api/transactions/tx1/handover/verify': invalid,
    })
    setup('/deals/tx1/complete#t=used')
    expect(await screen.findByRole('alert')).toHaveTextContent('この完了リンクは使えません。')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('「まだ受け取っていない」で閉じても、あとで「受け渡しを完了する」から確認し直せる', async () => {
    stubDealsFetch({
      'GET /api/transactions/tx1': () => json(buyerDealTransaction()),
      'POST /api/transactions/tx1/handover/verify': verified,
    })
    const { user } = setup('/deals/tx1/complete#t=link-token')
    await user.click(await screen.findByRole('button', { name: 'まだ受け取っていない' }))
    await user.click(screen.getByRole('button', { name: '受け渡しを完了する' }))
    expect(
      await screen.findByRole('dialog', { name: '商品を受け取りましたか？' }),
    ).toBeInTheDocument()
  })

  it('6桁のコードを入力して「次へ」で照合し、違えばコード欄にエラーを出す', async () => {
    let ok = false
    const fetchMock = stubDealsFetch({
      'GET /api/transactions/tx1': () => json(buyerDealTransaction()),
      'POST /api/transactions/tx1/handover/verify': () => (ok ? verified() : invalid()),
    })
    const { user } = setup('/deals/tx1/complete')
    await screen.findByRole('heading', { name: '完了コードを入力' })
    const next = screen.getByRole('button', { name: '次へ' })
    expect(next).toBeDisabled()

    await user.click(screen.getByRole('textbox', { name: '1桁目' }))
    await user.keyboard('582017')
    await user.click(next)
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'コードが違うか、有効期限が切れています',
    )
    expect(bodyOf(fetchMock, 'POST /api/transactions/tx1/handover/verify')).toEqual({
      kind: 'code',
      secret: '582017',
    })

    ok = true
    await user.click(next)
    expect(
      await screen.findByRole('dialog', { name: '商品を受け取りましたか？' }),
    ).toBeInTheDocument()
  })

  it('出品者が開いても、相手（購入者）の学籍番号を「出品者の学籍番号」として出さない', async () => {
    stubDealsFetch({
      'GET /api/transactions/tx1': () =>
        json(buyerDealTransaction({ role: 'seller', counterpartStudentNumber: '26141023' })),
    })
    setup('/deals/tx1/complete')
    expect(await screen.findByRole('alert')).toHaveTextContent(
      '受け渡しを完了できるのは、取引中の取引の購入者だけです。',
    )
    expect(screen.queryByText('26141023')).not.toBeInTheDocument()
  })

  it('完了済みの取引では入力欄を出さない', async () => {
    stubDealsFetch({
      'GET /api/transactions/tx1': () => json(buyerDealTransaction({ status: 'completed' })),
    })
    setup('/deals/tx1/complete')
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'この取引は受け渡しが完了しています。',
    )
    expect(screen.queryByRole('button', { name: '次へ' })).not.toBeInTheDocument()
  })
})
