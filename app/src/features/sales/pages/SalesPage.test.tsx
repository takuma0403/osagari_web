import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { stubDealsFetch } from '../../deals/test/fixtures'
import { json } from '../../items/test/fixtures'
import { renderRoute } from '../../../test/renderRoute'
import { sales, salesEntry } from '../test/fixtures'
import { SalesPage } from './SalesPage'

const salesPath = 'GET /api/me/sales'
const payoutPath = 'POST /api/me/sales/payouts'

/** 出金後の売上（確定していた売上が出金済みに移る） */
const paidOut = sales({
  availableAmount: 0,
  paidOutAmount: 4693,
  entries: [salesEntry({ state: 'paid_out', paidOutAt: '2026-10-27T03:00:00Z' })],
})

function setup() {
  return renderRoute('/mypage/sales', <SalesPage />)
}

describe('SalesPage', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('サーバーの集計と明細をそのまま出す', async () => {
    stubDealsFetch({ [salesPath]: () => json(sales()) })
    setup()
    const summary = await screen.findByRole('region', { name: '売上の集計' })
    expect(within(summary).getByText('¥1,473')).toBeInTheDocument()
    expect(within(summary).getByText('¥760')).toBeInTheDocument()
    expect(within(summary).getByText('¥3,220')).toBeInTheDocument()
    expect(within(summary).getByText('[銀行口座 未登録]')).toBeInTheDocument()

    const rows = within(screen.getByRole('list')).getAllByRole('listitem')
    expect(rows.map((r) => r.textContent)).toEqual([
      'ローテーブル 折りたたみ¥76010/8 · 受け渡し待ち¥800 − 手数料（5%） ¥40',
      '線形代数入門¥1,47310/5 · 確定¥1,550 − 手数料（5%） ¥77',
      'ミクロ経済学入門¥3,2209/20 · 出金済み¥3,500 − 手数料（8%） ¥280',
    ])
    expect(
      within(screen.getByRole('list'))
        .getAllByRole('link')
        .map((l) => l.getAttribute('href')),
    ).toEqual(['/items/def456', '/items/abc123', '/items/ghi789'])
    expect(screen.getByText('モック')).toBeInTheDocument()
  })

  it('売上がなければその旨を出し、出金は申請できない', async () => {
    stubDealsFetch({
      [salesPath]: () =>
        json(
          sales({ availableAmount: 0, awaitingHandoverAmount: 0, paidOutAmount: 0, entries: [] }),
        ),
    })
    setup()
    expect(await screen.findByText('まだ売上はありません。')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '出金を申請する' })).toBeDisabled()
  })

  it('確認してから出金を申請し、出金後の売上を出す', async () => {
    const fetchMock = stubDealsFetch({
      [salesPath]: () => json(sales()),
      [payoutPath]: () => json(paidOut),
    })
    setup()
    await userEvent.click(await screen.findByRole('button', { name: '出金を申請する' }))
    const dialog = screen.getByRole('dialog', { name: '出金を申請しますか？' })
    expect(within(dialog).getByText('¥1,473')).toBeInTheDocument()

    await userEvent.click(within(dialog).getByRole('button', { name: '出金を申請する' }))
    expect(await screen.findByText('出金を申請しました')).toBeInTheDocument()
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    const summary = screen.getByRole('region', { name: '売上の集計' })
    expect(within(summary).getByText('¥0')).toBeInTheDocument()
    expect(within(summary).getByText('¥4,693')).toBeInTheDocument()
    expect(within(summary).getByRole('button', { name: '出金を申請する' })).toBeDisabled()
    expect(fetchMock.mock.calls.filter(([, init]) => init?.method === 'POST')).toHaveLength(1)
  })

  it('確認でやめたら申請しない', async () => {
    const fetchMock = stubDealsFetch({ [salesPath]: () => json(sales()) })
    setup()
    await userEvent.click(await screen.findByRole('button', { name: '出金を申請する' }))
    await userEvent.click(screen.getByRole('button', { name: 'やめる' }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('出金できる売上がなくなっていた（409）ときは、エラーを出して売上を読み直す', async () => {
    let loads = 0
    stubDealsFetch({
      [salesPath]: () => json(++loads === 1 ? sales() : paidOut),
      [payoutPath]: () =>
        json({ code: 'invalid_status_transition', message: '出金できる売上がありません' }, 409),
    })
    setup()
    await userEvent.click(await screen.findByRole('button', { name: '出金を申請する' }))
    await userEvent.click(
      within(screen.getByRole('dialog')).getByRole('button', { name: '出金を申請する' }),
    )
    expect(await screen.findByRole('alert')).toHaveTextContent('出金できる売上がありません')
    expect(await screen.findByText('¥4,693')).toBeInTheDocument()
    expect(loads).toBe(2)
  })

  it('読み込めなかったら、もう一度読み込める', async () => {
    let loads = 0
    stubDealsFetch({
      [salesPath]: () =>
        ++loads === 1
          ? json({ code: 'internal', message: '読み込めませんでした' }, 500)
          : json(sales()),
    })
    setup()
    expect(await screen.findByRole('alert')).toHaveTextContent('読み込めませんでした')
    await userEvent.click(screen.getByRole('button', { name: 'もう一度読み込む' }))
    expect(await screen.findByRole('region', { name: '売上の集計' })).toBeInTheDocument()
  })
})
