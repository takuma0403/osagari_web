import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { onUnauthorized } from '../../api/client'
import { getMe, type CurrentUser } from './api'
import { SessionContext, type Session } from './session'

/**
 * ログイン状態を持つ（AUTH-1, AUTHN-5）。起動時に GET /me で確かめ、
 * どの API でも 401 unauthorized が返ったら未ログインに戻す（ガードがログイン画面へ送る）。
 */
export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<CurrentUser | null | undefined>(undefined)

  useEffect(() => {
    const controller = new AbortController()
    getMe(controller.signal)
      .then(setUser)
      // 401 のほか、通信の失敗なども未ログインとして扱う（ログイン画面からやり直せる）
      .catch(() => {
        if (!controller.signal.aborted) setUser(null)
      })
    return () => controller.abort()
  }, [])

  useEffect(() => onUnauthorized(() => setUser(null)), [])

  const session = useMemo<Session>(
    () => ({ user, signIn: setUser, signOut: () => setUser(null) }),
    [user],
  )
  return <SessionContext value={session}>{children}</SessionContext>
}
