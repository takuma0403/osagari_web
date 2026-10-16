import { Navigate, Outlet, useLocation } from 'react-router'
import { redirectTarget, type RedirectState } from './navigation'
import { useSession } from './session'

/**
 * ログインが必要な画面のガード（AUTH-1, AUTHN-5）。レイアウトルートとして使う。
 * 確認中は何も出さず、未ログインならログイン画面へ送る（開こうとしていたページを state に残す）。
 */
export function RequireAuth() {
  const { user } = useSession()
  const location = useLocation()
  if (user === undefined) return null
  if (user === null) {
    const state: RedirectState = { from: location.pathname + location.search + location.hash }
    return <Navigate to="/login" replace state={state} />
  }
  return <Outlet />
}

/**
 * ログイン前だけの画面（ログイン・新規登録・パスワード再設定・再確認）のガード。
 * ログイン済みなら、開こうとしていたページ（なければ商品一覧）へ送る。
 * 各画面はログインに成功したら signIn するだけで、行き先はここで決める。
 */
export function GuestOnly() {
  const { user } = useSession()
  const location = useLocation()
  if (user === undefined) return null
  if (user) return <Navigate to={redirectTarget(location.state)} replace />
  return <Outlet />
}
