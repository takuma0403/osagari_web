import { createContext, useContext } from 'react'
import type { CurrentUser } from './api'

export type Session = {
  /** ログイン中の本人。undefined は確認中（GET /me の応答待ち）、null は未ログイン */
  user: CurrentUser | null | undefined
  /** ログイン・再確認・新規登録・パスワード再設定に成功したときに、応答の本人の情報で呼ぶ */
  signIn: (user: CurrentUser) => void
  /** 未ログインに戻す（セッションが無効だったときなど） */
  signOut: () => void
}

export const SessionContext = createContext<Session | null>(null)

/** ログイン状態（AUTHN-5）。AuthProvider の内側で使う */
export function useSession(): Session {
  const session = useContext(SessionContext)
  if (!session) throw new Error('useSession は AuthProvider の内側で使ってください')
  return session
}
