import type { PasswordResetCodeVerified, SignupCodeVerified } from './api'

/**
 * 新規登録・パスワード再設定・再確認の画面間で location.state に入れて渡す値。
 * history.state に入るので再読み込みしても残り、URL・localStorage には載らない。
 */

/**
 * メールアドレスの入力 → コード入力（/signup → /signup/code、/password-reset → /password-reset/code）：
 * コードを送った学籍番号と送った時刻（ミリ秒）。有効期限と再送の待ち時間の起点
 */
export type CodeSentState = { studentId: string; sentAt: number }

/** /signup/code → /signup/password：パスワード設定用の一時トークンと推定結果 */
export type SignupPasswordState = SignupCodeVerified

/** /password-reset/code → /password-reset/password：パスワード設定用の一時トークン */
export type PasswordResetState = PasswordResetCodeVerified

const asRecord = (state: unknown): Record<string, unknown> | null =>
  typeof state === 'object' && state !== null ? (state as Record<string, unknown>) : null

export function isCodeSentState(state: unknown): state is CodeSentState {
  const s = asRecord(state)
  return typeof s?.studentId === 'string' && typeof s.sentAt === 'number'
}

export function isPasswordResetState(state: unknown): state is PasswordResetState {
  return typeof asRecord(state)?.setupToken === 'string'
}

export function isSignupPasswordState(state: unknown): state is SignupPasswordState {
  return isPasswordResetState(state) && asRecord(asRecord(state)?.estimate) !== null
}

/**
 * ガードがログイン画面へ送るときの state：開こうとしていたページ（パス＋クエリ＋ハッシュ）。
 * ログイン画面から再確認へも引き継ぎ、ログインできたらここへ戻す。
 */
export type RedirectState = { from?: string }

/** ログインできたときの行き先。開こうとしていたページがなければ商品一覧 */
export function redirectTarget(state: unknown): string {
  const from = asRecord(state)?.from
  // 外部のURLへは送らない（パスだけを受け付ける）
  return typeof from === 'string' && from.startsWith('/') && !from.startsWith('//') ? from : '/'
}
