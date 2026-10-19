import { postJson as post, request } from '../../api/client'
import type { components } from '../../api/schema.gen'
import { toUniversityEmail } from '../../lib/studentId'

export type StudentEstimate = components['schemas']['StudentEstimate']
export type SignupCodeVerified = components['schemas']['SignupCodeVerified']
export type LoginChallenge = components['schemas']['LoginChallenge']
export type CurrentUser = components['schemas']['CurrentUser']
export type PasswordResetCodeVerified = components['schemas']['PasswordResetCodeVerified']

/** 認証コードの有効期限（AUTHN-2） */
export const CODE_TTL_SECONDS = 10 * 60
/** 同じアドレスへのコード送信の間隔（AUTHN-6） */
export const CODE_RESEND_INTERVAL_SECONDS = 60

/** 学籍番号から学部・入学年度・有効期限を推定する（AUTH-2, AUTH-4）。登録済みかどうかには触れない */
export function estimateStudent(studentId: string, signal?: AbortSignal) {
  return post<StudentEstimate>(
    '/auth/signup/estimate',
    { email: toUniversityEmail(studentId) },
    signal,
  )
}

/** 新規登録の認証コードを送る（AUTHN-2）。登録済みでも同じ応答になる */
export function sendSignupCode(studentId: string) {
  return post<void>('/auth/signup/codes', { email: toUniversityEmail(studentId) })
}

/** 新規登録の認証コードを確認し、パスワード設定用の一時トークンを受け取る（AUTHN-2） */
export function verifySignupCode(studentId: string, code: string) {
  return post<SignupCodeVerified>('/auth/signup/codes/verify', {
    email: toUniversityEmail(studentId),
    code,
  })
}

/** パスワードを設定してアカウントを作り、そのままログインする（AUTHN-2, AUTHN-5） */
export function completeSignup(setupToken: string, password: string) {
  return post<CurrentUser>('/auth/signup/complete', { setupToken, password })
}

/** パスワード再設定の認証コードを送る（AUTHN-4）。未登録でも同じ応答になる */
export function sendPasswordResetCode(studentId: string) {
  return post<void>('/auth/password-reset/codes', { email: toUniversityEmail(studentId) })
}

/** パスワード再設定の認証コードを確認し、パスワード設定用の一時トークンを受け取る（AUTHN-4） */
export function verifyPasswordResetCode(studentId: string, code: string) {
  return post<PasswordResetCodeVerified>('/auth/password-reset/codes/verify', {
    email: toUniversityEmail(studentId),
    code,
  })
}

/** 新しいパスワードを設定し、ほかのセッションを無効にしてこの端末でログインする（AUTHN-4） */
export function completePasswordReset(setupToken: string, password: string) {
  return post<CurrentUser>('/auth/password-reset/complete', { setupToken, password })
}

/** ログインの再確認のコードを送り直す（AUTHN-3）。古いコードは使えなくなる */
export function resendRecheckCode(challengeToken: string) {
  return post<void>('/auth/login/recheck/codes', { challengeToken })
}

/** ログインの再確認のコードを確認してログインする（AUTHN-3） */
export function verifyRecheck(challengeToken: string, code: string) {
  return post<CurrentUser>('/auth/login/recheck/verify', { challengeToken, code })
}

/** ログイン中の本人の情報（AUTHN-5）。ログインしていなければ 401 unauthorized */
export function getMe(signal?: AbortSignal) {
  return request<CurrentUser>('/me', { signal })
}

export type LoginResult =
  { kind: 'loggedIn'; user: CurrentUser } | { kind: 'recheck'; challenge: LoginChallenge }

/**
 * メールアドレスとパスワードでログインする（AUTHN-3）。
 * 200 はそのままログイン、202 は再確認が必要（コードを送った）。request は status を返さないので、本文の形で見分ける。
 */
export async function login(studentId: string, password: string): Promise<LoginResult> {
  const body = await post<CurrentUser | LoginChallenge>('/auth/login', {
    email: toUniversityEmail(studentId),
    password,
  })
  return 'challengeToken' in body
    ? { kind: 'recheck', challenge: body }
    : { kind: 'loggedIn', user: body }
}
