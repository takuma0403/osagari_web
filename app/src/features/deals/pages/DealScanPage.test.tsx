import { act, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { StartScan } from '../../../lib/cameraScan'
import { renderRoute } from '../../../test/renderRoute'
import { json } from '../../items/test/fixtures'
import { approvedTransaction, bodyOf, buyerDealTransaction, stubDealsFetch } from '../test/fixtures'
import { DealScanPage } from './DealScanPage'

const qrOf = (dealId: string, token: string) =>
  `https://example.com/deals/${dealId}/complete#q=${token}`

/** カメラの代わり。emit で読み取った値を流す */
function fakeCamera() {
  let emit: (text: string) => void = () => {}
  const startScan: StartScan = vi.fn(async (_video, onText) => {
    emit = onText
    return { stop: vi.fn() }
  })
  return { startScan, emit: (text: string) => act(() => emit(text)) }
}

function setup(startScan: StartScan) {
  const view = renderRoute('/deals/:dealId/scan', <DealScanPage startScan={startScan} />, {
    at: '/deals/tx1/scan',
    destinations: ['/deals/:dealId', '/deals/:dealId/complete'],
  })
  return { ...view, user: userEvent.setup() }
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('DealScanPage', () => {
  it('この取引のQRを読み取ったら照合し、「受け取った」で完了して取引詳細へ進む', async () => {
    const camera = fakeCamera()
    const fetchMock = stubDealsFetch({
      'GET /api/transactions/tx1': () => json(buyerDealTransaction()),
      'POST /api/transactions/tx1/handover/verify': () => new Response(null, { status: 204 }),
      'POST /api/transactions/tx1/complete': () =>
        json(buyerDealTransaction({ status: 'completed', handoverMethod: 'qr' })),
    })
    const { user, router } = setup(camera.startScan)
    await screen.findByLabelText('カメラの映像')
    await act(async () => {})

    camera.emit(qrOf('tx1', 'qr-token'))
    const dialog = await screen.findByRole('dialog', { name: '商品を受け取りましたか？' })
    expect(dialog).toHaveTextContent('¥1,500')
    expect(dialog).toHaveTextContent('線形代数入門')
    expect(bodyOf(fetchMock, 'POST /api/transactions/tx1/handover/verify')).toEqual({
      kind: 'qr',
      secret: 'qr-token',
    })

    await user.click(screen.getByRole('button', { name: '受け取った' }))
    expect(bodyOf(fetchMock, 'POST /api/transactions/tx1/complete')).toEqual({
      kind: 'qr',
      secret: 'qr-token',
    })
    expect(router.state.location.pathname).toBe('/deals/tx1')
  })

  it('別の取引のQRは照合せず、案内を出す', async () => {
    const camera = fakeCamera()
    const fetchMock = stubDealsFetch({
      'GET /api/transactions/tx1': () => json(buyerDealTransaction()),
    })
    setup(camera.startScan)
    await screen.findByLabelText('カメラの映像')
    await act(async () => {})

    camera.emit(qrOf('tx2', 'qr-token'))
    expect(screen.getByRole('alert')).toHaveTextContent('この取引の受け渡しQRではありません')
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('使えないQRは案内を出し、同じQRを写し続けても照合し直さない', async () => {
    const camera = fakeCamera()
    const fetchMock = stubDealsFetch({
      'GET /api/transactions/tx1': () => json(buyerDealTransaction()),
      'POST /api/transactions/tx1/handover/verify': () =>
        json({ code: 'invalid_handover_token', message: '使えません' }, 400),
    })
    setup(camera.startScan)
    await screen.findByLabelText('カメラの映像')
    await act(async () => {})

    camera.emit(qrOf('tx1', 'old'))
    expect(await screen.findByRole('alert')).toHaveTextContent('このQRコードは使えません')
    camera.emit(qrOf('tx1', 'old'))
    const verifies = fetchMock.mock.calls.filter(([url]) => url.endsWith('/handover/verify'))
    expect(verifies).toHaveLength(1)
  })

  it('「まだ受け取っていない」では完了しない', async () => {
    const camera = fakeCamera()
    const fetchMock = stubDealsFetch({
      'GET /api/transactions/tx1': () => json(buyerDealTransaction()),
      'POST /api/transactions/tx1/handover/verify': () => new Response(null, { status: 204 }),
    })
    const { user } = setup(camera.startScan)
    await screen.findByLabelText('カメラの映像')
    await act(async () => {})

    camera.emit(qrOf('tx1', 'qr-token'))
    await user.click(await screen.findByRole('button', { name: 'まだ受け取っていない' }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(fetchMock.mock.calls.some(([url]) => url.endsWith('/complete'))).toBe(false)
  })

  it('カメラを使えないときは6桁のコードの入力を案内する', async () => {
    stubDealsFetch({ 'GET /api/transactions/tx1': () => json(buyerDealTransaction()) })
    setup(() => Promise.reject(new DOMException('denied', 'NotAllowedError')))
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'カメラの使用が許可されていません。ブラウザの設定で許可するか、6桁のコードを入力してください',
    )
    expect(screen.getByRole('link', { name: '6桁のコードを入力する' })).toHaveAttribute(
      'href',
      '/deals/tx1/complete',
    )
  })

  it('出品者はQRを読み取れない（カメラを起動しない）', async () => {
    const camera = fakeCamera()
    stubDealsFetch({ 'GET /api/transactions/tx1': () => json(approvedTransaction({ id: 'tx1' })) })
    setup(camera.startScan)
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'QRを読み取れるのは、取引中の取引の購入者だけです。',
    )
    expect(camera.startScan).not.toHaveBeenCalled()
  })
})
