import { useCallback, useEffect, useState } from 'react'
import { Outlet } from 'react-router'
import { isApiError } from '../../api/client'
import { Button } from '../../components/Button/Button'
import { InfoBox } from '../../components/InfoBox/InfoBox'
import { NotFoundPage } from '../../components/NotFoundPage/NotFoundPage'
import { MobileLayout } from '../../layouts/MobileLayout/MobileLayout'
import { getAdminSession } from './api'
import styles from './RequireAdmin.module.css'

type Check =
  | { state: 'checking' }
  | { state: 'admin' }
  | { state: 'notFound' }
  | { state: 'error'; message: string }

/**
 * 管理画面（/admin/**）のガード（ADMIN, FE, ADR 0015）。レイアウトルートとして RequireAuth の内側で使う。
 * GET /api/admin/session が成功したときだけ管理画面を出す。404 なら知らないパスと同じ「ページが見つかりません」を出し、
 * 管理画面の枠（管理メニュー）も出さない。確認中は何も出さない。
 */
export function RequireAdmin() {
  const [check, setCheck] = useState<Check>({ state: 'checking' })
  const [reloadKey, setReloadKey] = useState(0)

  useEffect(() => {
    const controller = new AbortController()
    getAdminSession(controller.signal)
      .then(() => setCheck({ state: 'admin' }))
      .catch((err: unknown) => {
        if (controller.signal.aborted) return
        if (isApiError(err) && err.code === 'not_found') setCheck({ state: 'notFound' })
        else
          setCheck({
            state: 'error',
            message: isApiError(err) ? err.message : '読み込めませんでした。もう一度お試しください',
          })
      })
    return () => controller.abort()
  }, [reloadKey])

  const reload = useCallback(() => {
    setCheck({ state: 'checking' })
    setReloadKey((key) => key + 1)
  }, [])

  switch (check.state) {
    case 'checking':
      return null
    case 'admin':
      return <Outlet />
    case 'notFound':
      return (
        <MobileLayout>
          <NotFoundPage />
        </MobileLayout>
      )
    case 'error':
      return (
        <MobileLayout>
          <div className={styles.error}>
            <InfoBox tone="danger" icon="alert" role="alert">
              <p>{check.message}</p>
              <Button variant="text" size="sm" onClick={reload}>
                もう一度読み込む
              </Button>
            </InfoBox>
          </div>
        </MobileLayout>
      )
  }
}
