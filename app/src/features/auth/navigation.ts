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

/**
 * ガードがログイン画面へ送るときの state：開こうとしていたページ（パス＋クエリ＋ハッシュ）。
 * ログイン画面から再確認へも引き継ぎ、ログインできたらここへ戻す。
 */
export type RedirectState = { from?: string }

/** ログインできたときの行き先。開こうとしていたページがなければ商品一覧 */
export function redirectTarget(state: unknown): string {
  if (typeof state !== 'object' || state === null) return '/'
  const from = (state as Record<string, unknown>).from
  // 外部のURLへは送らない（パスだけを受け付ける）
  return typeof from === 'string' && from.startsWith('/') && !from.startsWith('//') ? from : '/'
}
