import { formatMonthDay } from '../../lib/format'
import type { SalesEntry, SalesEntryState } from './api'

/** 売上の状態の表示名（PAY-5） */
export const salesStateLabels: Record<SalesEntryState, string> = {
  awaiting_handover: '受け渡し待ち',
  settled: '確定',
  paid_out: '出金済み',
}

/** 明細に出す日付。最後に動いた日時（出金・受け渡し完了・取引成立の順。API の並びと同じ）を `10/5` の形にする */
export function salesEntryDate(entry: SalesEntry): string {
  return formatMonthDay(entry.paidOutAt ?? entry.completedAt ?? entry.approvedAt)
}
