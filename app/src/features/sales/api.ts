import { request } from '../../api/client'
import type { components } from '../../api/schema.gen'

export type Sales = components['schemas']['Sales']
export type SalesEntry = components['schemas']['SalesEntry']
export type SalesEntryState = components['schemas']['SalesEntryState']

/** 本人が出品者として成立した取引（取引中・完了）の売上の明細と集計を取得する（PAY-5、ADR 0029） */
export function getSales(signal?: AbortSignal) {
  return request<Sales>('/me/sales', { signal })
}

/**
 * 出金できる売上をすべて出金する（PAY-5、JOBS-2）。フェーズ1はモックで、記録するだけで実際のお金は動かない。
 * 出金できる売上がなければ 409 invalid_status_transition。応答は出金後の売上。
 */
export function requestPayout() {
  return request<Sales>('/me/sales/payouts', { method: 'POST' })
}
