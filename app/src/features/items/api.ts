import { request } from '../../api/client'
import type { components, operations } from '../../api/schema.gen'
import { toSearchParams, type ItemSearchConditions } from './searchConditions'

export type ItemSummary = components['schemas']['ItemSummary']
export type ItemList = components['schemas']['ItemList']
export type ItemListQuery = NonNullable<operations['listItems']['parameters']['query']>
export type PublicItem = components['schemas']['PublicItem']

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

/**
 * 商品ページ（/items/:shareId）に出す商品を取得する（BROWSE）。取引中・取引完了は売り切れとして status 付きで返る。
 * 下書き・取り下げ・停止の商品や、出品者のアカウントが使えない商品は 404 not_found（ADR 0018）。
 */
export function getPublicItem(shareId: string, signal?: AbortSignal) {
  return request<PublicItem>(`/items/shared/${encodeURIComponent(shareId)}`, { signal })
}
