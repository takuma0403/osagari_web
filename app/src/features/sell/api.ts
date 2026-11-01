import { postJson, request } from '../../api/client'
import type { components } from '../../api/schema.gen'
import type { UploadContentType } from '../../lib/imageResize'

export type OwnItem = components['schemas']['OwnItem']
export type ItemInput = components['schemas']['ItemInput']
export type ItemCategory = components['schemas']['ItemCategory']
export type ItemCondition = components['schemas']['ItemCondition']
export type Campus = components['schemas']['Campus']
export type ImageUpload = components['schemas']['ImageUpload']
export type FeeRate = components['schemas']['FeeRate']

/** 商品写真のアップロード用の署名付きURLを発行する（ITEM, STORAGE）。URL は15分有効で、一度しか使えない */
export function createItemImageUpload(contentType: UploadContentType) {
  return postJson<ImageUpload>('/items/image-uploads', { contentType })
}

/** 商品を下書き（publish: false）か公開中（publish: true）として作る（ITEM） */
export function createItem(input: ItemInput, publish: boolean) {
  return postJson<OwnItem>('/items', { ...input, publish })
}

/** 自分の商品を編集用に取得する（ITEM）。ほかの人の商品は 404 */
export function getOwnItem(itemId: string, signal?: AbortSignal) {
  return request<OwnItem>(`/items/${encodeURIComponent(itemId)}`, { signal })
}

/** 商品のすべての項目を置き換える（ITEM。編集できるのは下書き・公開中・取り下げだけ） */
export function updateItem(itemId: string, input: ItemInput) {
  return request<OwnItem>(`/items/${encodeURIComponent(itemId)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
}

/** 下書きか取り下げの商品を公開する（ITEM） */
export function publishItem(itemId: string) {
  return postJson<OwnItem>(`/items/${encodeURIComponent(itemId)}/publish`, {})
}

/** 現在の手数料率を取得する（PAY-2）。受け取り見込みの表示に使う */
export function getFeeRate(signal?: AbortSignal) {
  return request<FeeRate>('/settings/fee-rate', { signal })
}
