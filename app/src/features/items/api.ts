import { request } from '../../api/client'
import type { components, operations } from '../../api/schema.gen'
import { toSearchParams, type ItemSearchConditions } from './searchConditions'

export type ItemSummary = components['schemas']['ItemSummary']
export type ItemList = components['schemas']['ItemList']
export type ItemListQuery = NonNullable<operations['listItems']['parameters']['query']>

/**
 * 公開中の商品を新着順に一覧・検索する（BROWSE）。続きは前の応答の nextCursor を cursor に渡して取る。
 * 条件が不正（下限が上限より大きいなど）なら 400 bad_request。
 */
export function listItems(conditions: ItemSearchConditions, cursor?: string, signal?: AbortSignal) {
  const params = toSearchParams(conditions)
  if (cursor) params.set('cursor', cursor)
  const query = params.toString()
  return request<ItemList>(query ? `/items?${query}` : '/items', { signal })
}
