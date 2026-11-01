import { act, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderRoute } from '../../../test/renderRoute'
import { json } from '../../items/test/fixtures'
import {
  approvedTransaction,
  bodyOf,
  buyerDealTransaction,
  stubDealsFetch,
  transaction,
} from '../test/fixtures'
import { DealDetailPage } from './DealDetailPage'

const NOW = Date.parse('2026-10-08T04:00:00Z')
const inMinutes = (minutes: number) => new Date(NOW + minutes * 60 * 1000).toISOString()

/** 受け渡しQRのトークン。有効期限はいまの時刻から5分（発行し直したときもそこから5分） */
const qrToken = (token = 'qr-token') =>
  json(
    {
      kind: 'qr',
      token,
      code: '381902',
      expiresAt: new Date(Date.now() + 5 * 60 * 1000).toISOString(),
    },
    201,
  )

function setup(at = '/deals/tx1') {
  const view = renderRoute('/deals/:dealId', <DealDetailPage />, {
    at,
    destinations: ['/deals/:dealId/scan', '/deals/:dealId/complete'],
  })
  return { ...view, user: userEvent.setup({ advanceTimers: vi.advanceTimersByTime }) }
}

beforeEach(() => {
  vi.useFakeTimers({ now: NOW, shouldAdvanceTime: true })
})

afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe('DealDetailPage', () => {
  it('出品者・取引中なら受け渡しQRを発行し、QRと確認コードと残り時間を出す', async () => {
    const fetchMock = stubDealsFetch({
      'GET /api/transactions/tx1': () => json(approvedTransaction({ id: 'tx1' })),
      'POST /api/transactions/tx1/handover-tokens': () => qrToken(),
    })
    setup()
    expect(await screen.findByRole('img', { name: '受け渡し用QRコード' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { level: 1, name: '受け渡し' })).toBeInTheDocument()
    expect(screen.getByText('381 902')).toBeInTheDocument()
    expect(screen.getByText('5:00')).toBeInTheDocument()
    expect(bodyOf(fetchMock, 'POST /api/transactions/tx1/handover-tokens')).toEqual({ kind: 'qr' })
  })

  it('期限が切れたら確認コードを消し、「表示し直す」で発行し直す', async () => {
    let count = 0
    const fetchMock = stubDealsFetch({
      'GET /api/transactions/tx1': () => json(approvedTransaction({ id: 'tx1' })),
      'POST /api/transactions/tx1/handover-tokens': () => qrToken(`qr-${++count}`),
    })
    const { user } = setup()
    await screen.findByText('381 902')
    await act(() => vi.advanceTimersByTimeAsync(5 * 60 * 1000))
    expect(screen.getByText('有効期限が切れました')).toBeInTheDocument()
    expect(screen.queryByText('381 902')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: '表示し直す' }))
    expect(await screen.findByText('381 902')).toBeInTheDocument()
    const issues = fetchMock.mock.calls.filter(([url]) => url.endsWith('/handover-tokens'))
    expect(issues).toHaveLength(2)
  })

  it('QRの表示中は5秒ごとに取り直し、購入者が完了したら完了と確定した売上を出す', async () => {
    let completed = false
    stubDealsFetch({
      'GET /api/transactions/tx1': () =>
        json(
          completed
            ? approvedTransaction({
                id: 'tx1',
                status: 'completed',
                item: { shareId: 'abc123', title: '線形代数入門', status: 'completed' },
                completedAt: inMinutes(1),
                handoverMethod: 'qr',
              })
            : approvedTransaction({ id: 'tx1' }),
        ),
      'POST /api/transactions/tx1/handover-tokens': () => qrToken(),
    })
    setup()
    await screen.findByText('381 902')
    completed = true
    await act(() => vi.advanceTimersByTimeAsync(5000))
    expect(
      await screen.findByRole('heading', { name: '受け渡しが完了しました' }),
    ).toBeInTheDocument()
    expect(screen.getByText('¥1,425')).toBeInTheDocument()
  })

  it('完了リンクを発行すると、トークンをフラグメントに載せたリンクを出してコピーできる', async () => {
    const fetchMock = stubDealsFetch({
      'GET /api/transactions/tx1': () => json(approvedTransaction({ id: 'tx1' })),
      'POST /api/transactions/tx1/handover-tokens': (init) =>
        JSON.parse(String(init?.body)).kind === 'link'
          ? json(
              { kind: 'link', token: 'link-token', code: null, expiresAt: inMinutes(24 * 60) },
              201,
            )
          : qrToken(),
    })
    const { user } = setup()
    await screen.findByText('381 902')
    await user.click(screen.getByRole('button', { name: '完了リンクを発行' }))

    expect(
      await screen.findByRole('heading', { name: '完了リンクを発行しました' }),
    ).toBeInTheDocument()
    const link = screen.getByLabelText('完了リンク')
    expect(link).toHaveValue(`${window.location.origin}/deals/tx1/complete#t=link-token`)
    expect(screen.getByText('24:00')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'コピー' }))
    expect(screen.getByRole('button', { name: 'コピーしました' })).toBeInTheDocument()
    await expect(navigator.clipboard.readText()).resolves.toBe(
      `${window.location.origin}/deals/tx1/complete#t=link-token`,
    )
    expect(bodyOf(fetchMock, 'POST /api/transactions/tx1/handover-tokens')).toEqual({ kind: 'qr' })

    await user.click(screen.getByRole('button', { name: '受け渡し画面に戻る' }))
    expect(screen.getByRole('img', { name: '受け渡し用QRコード' })).toBeInTheDocument()
  })

  it('完了リンクの表示中も取り直し、購入者が完了したら完了を出す', async () => {
    let completed = false
    stubDealsFetch({
      'GET /api/transactions/tx1': () =>
        json(
          approvedTransaction({
            id: 'tx1',
            status: completed ? 'completed' : 'in_deal',
            completedAt: completed ? inMinutes(1) : null,
          }),
        ),
      'POST /api/transactions/tx1/handover-tokens': (init) =>
        JSON.parse(String(init?.body)).kind === 'link'
          ? json(
              { kind: 'link', token: 'link-token', code: null, expiresAt: inMinutes(24 * 60) },
              201,
            )
          : qrToken(),
    })
    const { user } = setup()
    await screen.findByText('381 902')
    await user.click(screen.getByRole('button', { name: '完了リンクを発行' }))
    await screen.findByRole('heading', { name: '完了リンクを発行しました' })
    completed = true
    await act(() => vi.advanceTimersByTimeAsync(5000))
    expect(
      await screen.findByRole('heading', { name: '受け渡しが完了しました' }),
    ).toBeInTheDocument()
  })

  it('完了コードを発行すると、6桁のコードを3桁ずつ区切って出す', async () => {
    stubDealsFetch({
      'GET /api/transactions/tx1': () => json(approvedTransaction({ id: 'tx1' })),
      'POST /api/transactions/tx1/handover-tokens': (init) =>
        JSON.parse(String(init?.body)).kind === 'code'
          ? json({ kind: 'code', token: null, code: '582017', expiresAt: inMinutes(24 * 60) }, 201)
          : qrToken(),
    })
    const { user } = setup()
    await screen.findByText('381 902')
    await user.click(screen.getByRole('button', { name: '完了コードを発行' }))
    expect(
      await screen.findByRole('heading', { name: '完了コードを発行しました' }),
    ).toBeInTheDocument()
    expect(screen.getByText('582 017')).toBeInTheDocument()
  })

  it('購入者・取引中なら、出品者の学籍番号と、QRの読み取り・コードの入力への導線を出す（QRは発行しない）', async () => {
    const fetchMock = stubDealsFetch({
      'GET /api/transactions/tx1': () => json(buyerDealTransaction()),
    })
    const { user, router } = setup()
    expect(await screen.findByText('25238117')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: '完了コードを入力' })).toHaveAttribute(
      'href',
      '/deals/tx1/complete',
    )
    await user.click(screen.getByRole('link', { name: 'QRを読み取る' }))
    expect(router.state.location.pathname).toBe('/deals/tx1/scan')
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  describe('キャンセル（DEAL-4）', () => {
    const cancelled = (overrides: Parameters<typeof buyerDealTransaction>[0] = {}) =>
      buyerDealTransaction({
        status: 'cancelled',
        item: { shareId: 'abc123', title: '線形代数入門', status: 'published' },
        buyerCancelAgreedAt: inMinutes(1),
        sellerCancelAgreedAt: inMinutes(-10),
        ...overrides,
      })

    it('出品者がリンクから申請すると、申請中の案内を出し、取り下げるとリンクに戻る', async () => {
      const fetchMock = stubDealsFetch({
        'GET /api/transactions/tx1': () => json(approvedTransaction({ id: 'tx1' })),
        'POST /api/transactions/tx1/handover-tokens': () => qrToken(),
        'POST /api/transactions/tx1/cancel': () =>
          json(approvedTransaction({ id: 'tx1', sellerCancelAgreedAt: inMinutes(0) })),
        'POST /api/transactions/tx1/cancel/withdraw': () =>
          json(approvedTransaction({ id: 'tx1' })),
      })
      const { user } = setup()
      await screen.findByText('381 902')
      await user.click(
        screen.getByRole('button', { name: '取引のキャンセルを申請（相手の同意が必要）' }),
      )
      const sheet = screen.getByRole('dialog', { name: '取引のキャンセルを申請' })
      expect(sheet).toHaveTextContent('¥1,500')
      await user.click(screen.getByRole('button', { name: 'キャンセルを申請する' }))

      expect(
        await screen.findByText('キャンセルを申請しました。相手が同意するまで取引は続いています。'),
      ).toBeInTheDocument()
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
      expect(
        screen.queryByRole('button', { name: '取引のキャンセルを申請（相手の同意が必要）' }),
      ).not.toBeInTheDocument()
      // 申請中も受け渡しは続けられる（ADR 0026）
      expect(screen.getByRole('img', { name: '受け渡し用QRコード' })).toBeInTheDocument()

      await user.click(screen.getByRole('button', { name: '申請を取り下げる' }))
      expect(
        await screen.findByRole('button', { name: '取引のキャンセルを申請（相手の同意が必要）' }),
      ).toBeInTheDocument()
      expect(bodyOf(fetchMock, 'POST /api/transactions/tx1/cancel')).toEqual({})
      expect(bodyOf(fetchMock, 'POST /api/transactions/tx1/cancel/withdraw')).toEqual({})
    })

    it('「やめる」で閉じ、申請しない', async () => {
      const fetchMock = stubDealsFetch({
        'GET /api/transactions/tx1': () => json(buyerDealTransaction()),
      })
      const { user } = setup()
      await user.click(
        await screen.findByRole('button', { name: '取引のキャンセルを申請（相手の同意が必要）' }),
      )
      await user.click(screen.getByRole('button', { name: 'やめる' }))
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
      expect(fetchMock).toHaveBeenCalledTimes(1)
    })

    it('相手の申請が届いていたらシートを開き、「同意しない」で閉じても案内から開き直せる', async () => {
      const fetchMock = stubDealsFetch({
        'GET /api/transactions/tx1': () =>
          json(buyerDealTransaction({ sellerCancelAgreedAt: inMinutes(-10) })),
      })
      const { user } = setup()
      expect(
        await screen.findByRole('dialog', { name: 'キャンセルの申請が届いています' }),
      ).toBeInTheDocument()
      await user.click(screen.getByRole('button', { name: '同意しない' }))
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
      expect(fetchMock).toHaveBeenCalledTimes(1)
      // 申請された側は申請のリンクを出さず、受け渡しの導線はそのまま
      expect(
        screen.queryByRole('button', { name: '取引のキャンセルを申請（相手の同意が必要）' }),
      ).not.toBeInTheDocument()
      expect(screen.getByRole('link', { name: 'QRを読み取る' })).toBeInTheDocument()

      await user.click(screen.getByRole('button', { name: '内容を確認する' }))
      expect(
        screen.getByRole('dialog', { name: 'キャンセルの申請が届いています' }),
      ).toBeInTheDocument()
    })

    it('出品者が同意シートを閉じたら、完了リンクの画面から戻っても開き直さない', async () => {
      stubDealsFetch({
        'GET /api/transactions/tx1': () =>
          json(approvedTransaction({ id: 'tx1', buyerCancelAgreedAt: inMinutes(-10) })),
        'POST /api/transactions/tx1/handover-tokens': (init) =>
          JSON.parse(String(init?.body)).kind === 'link'
            ? json(
                { kind: 'link', token: 'link-token', code: null, expiresAt: inMinutes(24 * 60) },
                201,
              )
            : qrToken(),
      })
      const { user } = setup()
      await screen.findByRole('dialog', { name: 'キャンセルの申請が届いています' })
      await user.click(screen.getByRole('button', { name: '同意しない' }))
      await user.click(await screen.findByRole('button', { name: '完了リンクを発行' }))
      await screen.findByRole('heading', { name: '完了リンクを発行しました' })
      // 発行画面でも申請が届いていることは出したまま
      expect(screen.getByRole('button', { name: '内容を確認する' })).toBeInTheDocument()
      await user.click(screen.getByRole('button', { name: '受け渡し画面に戻る' }))
      await screen.findByText('381 902')
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })

    it('同意すると、キャンセルと返金の表示になる', async () => {
      const fetchMock = stubDealsFetch({
        'GET /api/transactions/tx1': () =>
          json(buyerDealTransaction({ sellerCancelAgreedAt: inMinutes(-10) })),
        'POST /api/transactions/tx1/cancel': () => json(cancelled()),
      })
      const { user } = setup()
      await screen.findByRole('dialog', { name: 'キャンセルの申請が届いています' })
      await user.click(screen.getByRole('button', { name: 'キャンセルに同意する' }))
      expect(
        await screen.findByText('この取引はキャンセルされました。代金は購入者に返金されました。'),
      ).toBeInTheDocument()
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
      expect(screen.queryByRole('link', { name: 'QRを読み取る' })).not.toBeInTheDocument()
      expect(fetchMock).toHaveBeenCalledTimes(2)
    })

    it('取引中でなくなっていたら（409）エラーを出して取り直す', async () => {
      let count = 0
      stubDealsFetch({
        'GET /api/transactions/tx1': () =>
          json(
            ++count === 1
              ? buyerDealTransaction({ sellerCancelAgreedAt: inMinutes(-10) })
              : buyerDealTransaction({
                  status: 'completed',
                  item: { shareId: 'abc123', title: '線形代数入門', status: 'completed' },
                  completedAt: inMinutes(1),
                }),
          ),
        'POST /api/transactions/tx1/cancel': () =>
          json(
            { code: 'invalid_status_transition', message: 'この取引はキャンセルできません' },
            409,
          ),
      })
      const { user } = setup()
      await screen.findByRole('dialog', { name: 'キャンセルの申請が届いています' })
      await user.click(screen.getByRole('button', { name: 'キャンセルに同意する' }))
      expect(
        await screen.findByRole('heading', { name: '受け渡しが完了しました' }),
      ).toBeInTheDocument()
      expect(count).toBe(2)
    })
  })

  it('取引中でない取引は、ステータスの説明だけを出す', async () => {
    stubDealsFetch({ 'GET /api/transactions/tx1': () => json(transaction()) })
    setup()
    expect(await screen.findByText('出品者の承認を待っています。')).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'QRを読み取る' })).not.toBeInTheDocument()
  })

  it('当事者でない取引は「取引が見つかりません」', async () => {
    stubDealsFetch({
      'GET /api/transactions/tx1': () =>
        json({ code: 'not_found', message: '見つかりません' }, 404),
    })
    setup()
    expect(await screen.findByRole('heading', { name: '取引が見つかりません' })).toBeInTheDocument()
  })
})
