import { transactionListEntry } from '../deals/test/fixtures'
import type { OwnItemSummary } from './api'
import {
  isPurchased,
  ownItemDisplay,
  ownStatusLabels,
  presentStatuses,
  purchaseDisplay,
  sortByUpdatedAt,
} from './myPageDisplay'

function ownItem(overrides: Partial<OwnItemSummary> = {}): OwnItemSummary {
  return {
    id: 'i1',
    shareId: 'abc123',
    status: 'published',
    title: '線形代数入門',
    price: 1500,
    thumbnail: null,
    requestCount: 0,
    updatedAt: '2026-10-07T03:40:00Z',
    ...overrides,
  }
}

describe('ownItemDisplay', () => {
  it.each([
    ['requested', 3, '/sell/i1/requests', '購入希望 3件', 'warning'],
    ['draft', 0, '/sell/i1', '下書き', 'neutral'],
    ['withdrawn', 0, '/sell/i1', '取り下げ', 'neutral'],
    ['suspended', 0, '/sell/i1', '停止中', 'danger'],
    ['published', 0, '/items/abc123', '公開中', 'primary'],
    ['in_deal', 0, '/items/abc123', '取引中', 'primary'],
    ['completed', 0, '/items/abc123', '売却済み', 'neutral'],
  ] as const)('%s は %s へ進み「%s」を出す', (status, requestCount, to, label, tone) => {
    expect(ownItemDisplay(ownItem({ status, requestCount }))).toEqual({ to, status: label, tone })
  })
})

describe('isPurchased', () => {
  it.each([
    ['in_deal', true],
    ['completed', true],
    ['requested', false],
    ['rejected', false],
    ['expired', false],
    ['cancelled', false],
  ] as const)('%s は %s', (status, expected) => {
    expect(isPurchased(transactionListEntry({ status }))).toBe(expected)
  })
})

describe('purchaseDisplay', () => {
  it('取引一覧の購入側と同じバッジで取引詳細へ進む', () => {
    expect(
      purchaseDisplay(transactionListEntry({ transactionId: 'tx9', status: 'completed' })),
    ).toEqual({ to: '/deals/tx9', status: '完了', tone: 'neutral' })
  })
})

describe('presentStatuses', () => {
  it('一覧にある状態だけを、名前の表の順に返す', () => {
    const list = [
      ownItem({ status: 'draft' }),
      ownItem({ status: 'published' }),
      ownItem({ status: 'draft' }),
    ]
    expect(presentStatuses(ownStatusLabels, list)).toEqual(['published', 'draft'])
  })
})

describe('sortByUpdatedAt', () => {
  const a = ownItem({ id: 'a', updatedAt: '2026-10-01T00:00:00Z' })
  const b = ownItem({ id: 'b', updatedAt: '2026-10-03T00:00:00Z' })
  const c = ownItem({ id: 'c', updatedAt: '2026-10-02T00:00:00Z' })

  it('新しい順と古い順に並べ、元の配列は変えない', () => {
    const list = [a, b, c]
    expect(sortByUpdatedAt(list, 'desc').map((x) => x.id)).toEqual(['b', 'c', 'a'])
    expect(sortByUpdatedAt(list, 'asc').map((x) => x.id)).toEqual(['a', 'c', 'b'])
    expect(list.map((x) => x.id)).toEqual(['a', 'b', 'c'])
  })
})
