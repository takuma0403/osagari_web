import { request } from '../../api/client'
import type { components } from '../../api/schema.gen'

export type AdminStudentCard = components['schemas']['AdminStudentCard']
export type StudentCardImage = components['schemas']['StudentCardImage']
export type StudentCardView = components['schemas']['StudentCardView']
export type AdminUser = components['schemas']['AdminUser']
export type AdminUserFilter = components['schemas']['AdminUserFilter']
export type AdminUserList = components['schemas']['AdminUserList']
export type AdminItem = components['schemas']['AdminItem']
export type AdminItemList = components['schemas']['AdminItemList']
export type AdminTransaction = components['schemas']['AdminTransaction']
export type AdminTransactionList = components['schemas']['AdminTransactionList']
export type AdminFeeRate = components['schemas']['AdminFeeRate']

/** 管理者かどうかを確かめる（ADMIN, FE）。管理者なら成功し、未ログイン・管理者でなければ 404 not_found（ADR 0015） */
export function getAdminSession(signal?: AbortSignal) {
  return request<void>('/admin/session', { signal })
}

/** 審査待ちの学生証を提出の古い順に一覧する（ADMIN, CARD）。画像の URL は含まない */
export function listPendingStudentCards(signal?: AbortSignal) {
  return request<AdminStudentCard[]>('/admin/student-cards', { signal })
}

/** 学生証画像の閲覧URL（15分有効）を発行する（ADMIN, CARD, STORAGE）。発行するたびに閲覧が記録される */
export function createStudentCardView(studentCardId: string, signal?: AbortSignal) {
  return request<StudentCardImage>(
    `/admin/student-cards/${encodeURIComponent(studentCardId)}/views`,
    {
      method: 'POST',
      signal,
    },
  )
}

/** 学生証画像の閲覧記録を新しい順に一覧する（ADMIN, CARD） */
export function listStudentCardViews(studentCardId: string, signal?: AbortSignal) {
  return request<StudentCardView[]>(
    `/admin/student-cards/${encodeURIComponent(studentCardId)}/views`,
    { signal },
  )
}

/** 学生証を承認し、本人を出品できる状態にする（ADMIN, CARD） */
export function approveStudentCard(studentCardId: string) {
  return request<AdminStudentCard>(
    `/admin/student-cards/${encodeURIComponent(studentCardId)}/approve`,
    { method: 'POST' },
  )
}

/** 学生証を差し戻す（ADMIN, CARD）。理由は持たない（ADR 0015） */
export function rejectStudentCard(studentCardId: string) {
  return request<AdminStudentCard>(
    `/admin/student-cards/${encodeURIComponent(studentCardId)}/reject`,
    { method: 'POST' },
  )
}

/** 一覧の続きを取る位置を付けたパスにする（cursor が無ければ先頭から） */
function withQuery(path: string, params: URLSearchParams) {
  const query = params.toString()
  return query ? `${path}?${query}` : path
}

/**
 * ユーザーを登録の新しい順に一覧する（ADMIN, AUTH-3）。q は学籍番号の前方一致、filter は状態での絞り込み（ADR 0034）。
 * 続きは前の応答の nextCursor を cursor に渡して取る
 */
export function listAdminUsers(
  conditions: { q?: string; filter?: AdminUserFilter },
  cursor?: string,
  signal?: AbortSignal,
) {
  const params = new URLSearchParams()
  if (conditions.q) params.set('q', conditions.q)
  if (conditions.filter && conditions.filter !== 'all') params.set('filter', conditions.filter)
  if (cursor) params.set('cursor', cursor)
  return request<AdminUserList>(withQuery('/admin/users', params), { signal })
}

const userPath = (userId: string) => `/admin/users/${encodeURIComponent(userId)}`

/** アカウントを停止し、セッションをすべて無効にする（ADMIN、ADR 0034）。有効でなければ 409 */
export function suspendUser(userId: string) {
  return request<AdminUser>(`${userPath(userId)}/suspend`, { method: 'POST' })
}

/** アカウントの停止を解除する（ADMIN、ADR 0034）。停止中でなければ 409 */
export function unsuspendUser(userId: string) {
  return request<AdminUser>(`${userPath(userId)}/unsuspend`, { method: 'POST' })
}

/** 有効期限を変える（ADMIN, AUTH-3。延長の仕組みができるまでの暫定）。今日より前の日付なら 400 */
export function updateUserExpiresOn(userId: string, expiresOn: string) {
  return request<AdminUser>(`${userPath(userId)}/expires-on`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ expiresOn }),
  })
}

/** 退学によりアカウントを削除する（ADMIN, AUTH-3）。失効と同じ後処理を行う（ADR 0033, ADR 0034） */
export function deleteUser(userId: string) {
  return request<AdminUser>(`${userPath(userId)}/delete`, { method: 'POST' })
}

/** すべての出品を作った日時の新しい順に一覧する（ADMIN）。続きは nextCursor を cursor に渡して取る */
export function listAdminItems(cursor?: string, signal?: AbortSignal) {
  const params = new URLSearchParams()
  if (cursor) params.set('cursor', cursor)
  return request<AdminItemList>(withQuery('/admin/items', params), { signal })
}

/** 出品を停止する（ADMIN, ITEM-1）。購入希望ありなら承認待ちの購入希望を却下する（ADR 0034） */
export function suspendItem(itemId: string) {
  return request<AdminItem>(`/admin/items/${encodeURIComponent(itemId)}/suspend`, {
    method: 'POST',
  })
}

/** すべての取引を最後に動いた日時の新しい順に一覧する（ADMIN）。続きは nextCursor を cursor に渡して取る */
export function listAdminTransactions(cursor?: string, signal?: AbortSignal) {
  const params = new URLSearchParams()
  if (cursor) params.set('cursor', cursor)
  return request<AdminTransactionList>(withQuery('/admin/transactions', params), { signal })
}

/** 取引中の取引をキャンセルし、代金を購入者に返金する（モック。ADMIN, DEAL-4、ADR 0034） */
export function cancelTransactionByAdmin(transactionId: string) {
  return request<AdminTransaction>(
    `/admin/transactions/${encodeURIComponent(transactionId)}/cancel`,
    { method: 'POST' },
  )
}

/** 現在の手数料率と、最後に変更した日時・管理者を取る（ADMIN, PAY-2） */
export function getAdminFeeRate(signal?: AbortSignal) {
  return request<AdminFeeRate>('/admin/settings/fee-rate', { signal })
}

/** 手数料率（ベーシスポイント）を変える（ADMIN, PAY-2）。変更後に成立する取引から使われる（ADR 0003） */
export function updateFeeRate(feeRateBasisPoints: number) {
  return request<AdminFeeRate>('/admin/settings/fee-rate', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ feeRateBasisPoints }),
  })
}
