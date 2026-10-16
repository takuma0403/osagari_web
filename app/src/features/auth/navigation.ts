import type { SignupCodeVerified } from './api'

/**
 * 新規登録の画面間で location.state に入れて渡す値。
 * history.state に入るので再読み込みしても残り、URL・localStorage には載らない。
 */

/** /signup → /signup/code：コードを送った学籍番号と送った時刻（ミリ秒）。有効期限と再送の待ち時間の起点 */
export type SignupCodeState = { studentId: string; sentAt: number }

/** /signup/code → /signup/password：パスワード設定用の一時トークンと推定結果（#14 で受け取る） */
export type SignupPasswordState = SignupCodeVerified

export function isSignupCodeState(state: unknown): state is SignupCodeState {
  if (typeof state !== 'object' || state === null) return false
  const s = state as Record<string, unknown>
  return typeof s.studentId === 'string' && typeof s.sentAt === 'number'
}
