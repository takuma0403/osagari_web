import type { Sales, SalesEntry } from '../api'

/** 売上の明細の1行（既定は受け渡しが完了して未出金の確定した売上） */
export function salesEntry(overrides: Partial<SalesEntry> = {}): SalesEntry {
  return {
    transactionId: 'tx1',
    item: { shareId: 'abc123', title: '線形代数入門', category: 'textbook', thumbnail: null },
    state: 'settled',
    price: 1550,
    feeRateBasisPoints: 500,
    fee: 77,
    sellerProceeds: 1473,
    approvedAt: '2026-10-01T03:00:00Z',
    completedAt: '2026-10-05T03:00:00Z',
    paidOutAt: null,
    ...overrides,
  }
}

/** 売上管理の集計と明細（確定・受け渡し待ち・出金済みが1件ずつ。出金済みは料率が 8% のときに成立した取引） */
export function sales(overrides: Partial<Sales> = {}): Sales {
  return {
    availableAmount: 1473,
    awaitingHandoverAmount: 760,
    paidOutAmount: 3220,
    entries: [
      salesEntry({
        transactionId: 'tx2',
        item: {
          shareId: 'def456',
          title: 'ローテーブル 折りたたみ',
          category: 'furniture_appliance',
          thumbnail: null,
        },
        state: 'awaiting_handover',
        price: 800,
        fee: 40,
        sellerProceeds: 760,
        approvedAt: '2026-10-08T03:00:00Z',
        completedAt: null,
      }),
      salesEntry(),
      salesEntry({
        transactionId: 'tx3',
        item: {
          shareId: 'ghi789',
          title: 'ミクロ経済学入門',
          category: 'textbook',
          thumbnail: null,
        },
        state: 'paid_out',
        price: 3500,
        feeRateBasisPoints: 800,
        fee: 280,
        sellerProceeds: 3220,
        approvedAt: '2026-09-15T03:00:00Z',
        completedAt: '2026-09-18T03:00:00Z',
        paidOutAt: '2026-09-20T03:00:00Z',
      }),
    ],
    ...overrides,
  }
}
