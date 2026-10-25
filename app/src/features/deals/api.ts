import { postJson, request } from '../../api/client'
import type { components } from '../../api/schema.gen'

export type Transaction = components['schemas']['Transaction']
export type PurchaseRequest = components['schemas']['PurchaseRequest']
export type PurchaseRequestList = components['schemas']['PurchaseRequestList']
export type PaymentMethod = components['schemas']['PaymentMethod']

const transactionPath = (transactionId: string) =>
  `/transactions/${encodeURIComponent(transactionId)}`

/**
 * 商品ページの商品に購入希望を出す（DEAL-1）。支払い方法の枠の確保はフェーズ1ではモックで記録するだけ（PAY）。
 * 自分の商品は 403 forbidden、売り切れや承認待ちの購入希望を出し済みなら 409 invalid_status_transition。
 */
export function createPurchaseRequest(shareId: string, paymentMethod: PaymentMethod) {
  return postJson<Transaction>(`/items/shared/${encodeURIComponent(shareId)}/requests`, {
    paymentMethod,
  })
}

/** 自分の商品に届いている承認待ちの購入希望を、希望の古い順に取得する（DEAL-1）。購入者の学籍番号は含まない */
export function listPurchaseRequests(itemId: string, signal?: AbortSignal) {
  return request<PurchaseRequestList>(`/items/${encodeURIComponent(itemId)}/requests`, { signal })
}

/** 購入希望を承認して取引を成立させる（DEAL-1）。応答には購入者の学籍番号が入る */
export function approveTransaction(transactionId: string) {
  return postJson<Transaction>(`${transactionPath(transactionId)}/approve`, {})
}

/** 購入希望を却下して枠を解放する（DEAL-1） */
export function rejectTransaction(transactionId: string) {
  return postJson<Transaction>(`${transactionPath(transactionId)}/reject`, {})
}

/** 取引成立の確認モーダルで「確認しました」を押した日時を記録する（DEAL-2） */
export function acknowledgeTransaction(transactionId: string) {
  return postJson<Transaction>(`${transactionPath(transactionId)}/acknowledge`, {})
}

export type HandoverTokenKind = components['schemas']['HandoverTokenKind']
export type HandoverToken = components['schemas']['HandoverToken']
export type HandoverCredential = components['schemas']['HandoverCredential']

/** 自分が購入者か出品者の取引を取得する（DEAL-1〜3）。当事者でなければ 404 */
export function getTransaction(transactionId: string, signal?: AbortSignal) {
  return request<Transaction>(transactionPath(transactionId), { signal })
}

/**
 * 出品者が受け渡しQR・完了リンク・完了コードのトークンを発行する（DEAL-3）。
 * 同じ種類の未使用のものは使えなくなる。qr は5分、link・code は24時間で期限切れ
 */
export function createHandoverToken(transactionId: string, kind: HandoverTokenKind) {
  return postJson<HandoverToken>(`${transactionPath(transactionId)}/handover-tokens`, { kind })
}

/** 購入者が読み取った・開いた・入力したトークンが使えるかを、使用済みにせずに確かめる（DEAL-3）。使えなければ 400 invalid_handover_token */
export function verifyHandover(transactionId: string, credential: HandoverCredential) {
  return postJson<void>(`${transactionPath(transactionId)}/handover/verify`, credential)
}

/** 購入者の「受け取った」で受け渡しを完了し、出品者の売上を確定する（DEAL-3、PAY-2） */
export function completeTransaction(transactionId: string, credential: HandoverCredential) {
  return postJson<Transaction>(`${transactionPath(transactionId)}/complete`, credential)
}
