import {
  cancelRequestText,
  isItemSuspendable,
  parseFeeRatePercent,
  transactionStatusDisplay,
} from './adminDisplay'
import type { AdminItem, AdminTransaction } from './api'

const tx = (overrides: Partial<AdminTransaction>): AdminTransaction => ({
  id: 't1',
  status: 'in_deal',
  item: { shareId: 's1', title: '線形代数入門', status: 'in_deal' },
  sellerStudentNumber: '25238117',
  buyerStudentNumber: '26141023',
  price: 1500,
  paymentMethod: 'card',
  feeRateBasisPoints: 500,
  buyerCancelAgreedAt: null,
  sellerCancelAgreedAt: null,
  completedAt: null,
  createdAt: '2026-10-08T05:02:00Z',
  updatedAt: '2026-10-08T05:02:00Z',
  ...overrides,
})

describe('transactionStatusDisplay', () => {
  it('取引中で片方が同意していればキャンセル申請中', () => {
    expect(transactionStatusDisplay(tx({ buyerCancelAgreedAt: '2026-10-08T05:02:00Z' }))).toEqual({
      label: 'キャンセル申請中',
      tone: 'warning',
    })
  })

  it.each([
    ['in_deal', '取引中'],
    ['requested', '購入希望'],
    ['completed', '完了'],
    ['cancelled', 'キャンセル'],
  ] as const)('%s は %s', (status, label) => {
    expect(transactionStatusDisplay(tx({ status })).label).toBe(label)
  })
})

describe('cancelRequestText', () => {
  const at = '2026-10-08T05:02:00Z'
  it.each([
    [{}, 'なし'],
    [{ buyerCancelAgreedAt: at }, '購入者が申請（出品者は未同意）'],
    [{ sellerCancelAgreedAt: at }, '出品者が申請（購入者は未同意）'],
    [{ buyerCancelAgreedAt: at, sellerCancelAgreedAt: at }, '双方が同意'],
  ])('%o は %s', (overrides, text) => {
    expect(cancelRequestText(tx(overrides))).toBe(text)
  })
})

describe('isItemSuspendable', () => {
  it.each([
    ['draft', true],
    ['published', true],
    ['requested', true],
    ['in_deal', true],
    ['completed', false],
    ['withdrawn', false],
    ['suspended', false],
  ] as const)('%s は %s', (status, expected) => {
    expect(isItemSuspendable({ status } as AdminItem)).toBe(expected)
  })
})

describe('parseFeeRatePercent', () => {
  it.each([
    ['5', 500],
    ['5.25', 525],
    ['0.1', 10],
    ['0', 0],
    ['100', 10000],
    [' 7.5 ', 750],
  ])('%s%% は %i ベーシスポイント', (input, expected) => {
    expect(parseFeeRatePercent(input)).toBe(expected)
  })

  it.each(['', '120', '100.01', '5.255', '-1', 'abc', '5.', '.5'])('%s は不正', (input) => {
    expect(parseFeeRatePercent(input)).toBeNull()
  })
})
