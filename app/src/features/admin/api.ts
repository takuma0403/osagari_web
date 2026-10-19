import { request } from '../../api/client'
import type { components } from '../../api/schema.gen'

export type AdminStudentCard = components['schemas']['AdminStudentCard']
export type StudentCardImage = components['schemas']['StudentCardImage']
export type StudentCardView = components['schemas']['StudentCardView']

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
