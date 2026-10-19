import { postJson, request } from '../../api/client'
import type { components } from '../../api/schema.gen'

export type StudentCardState = components['schemas']['StudentCardState']
export type StudentCardUpload = components['schemas']['StudentCardUpload']
export type StudentCardContentType = components['schemas']['StudentCardUploadInput']['contentType']

/** 学生証画像のアップロード用の署名付きURLを発行する（CARD, STORAGE）。URL は15分有効で、一度しか使えない */
export function createStudentCardUpload(contentType: StudentCardContentType) {
  return postJson<StudentCardUpload>('/student-cards/uploads', { contentType })
}

/** アップロードした学生証画像を登録して審査待ちにする（CARD, STORAGE） */
export function submitStudentCard(objectKey: string) {
  return postJson<StudentCardState>('/student-cards', { objectKey })
}

/** 本人の学生証の審査状況を取得する（CARD）。画像の閲覧URLは返らない（STORAGE） */
export function getCurrentStudentCard(signal?: AbortSignal) {
  return request<StudentCardState>('/student-cards/current', { signal })
}
