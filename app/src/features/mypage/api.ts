import { request } from '../../api/client'
import type { components } from '../../api/schema.gen'

export type OwnItemList = components['schemas']['OwnItemList']
export type OwnItemSummary = components['schemas']['OwnItemSummary']

/** 自分のすべての出品（下書き・取り下げ・停止を含む）を、最後に更新した順に取得する（AUTH-3、ADR 0029） */
export function listOwnItems(signal?: AbortSignal) {
  return request<OwnItemList>('/me/items', { signal })
}
