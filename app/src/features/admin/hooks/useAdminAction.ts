import { useCallback, useEffect, useState } from 'react'
import { isApiError } from '../../../api/client'
import { messageOf } from '../errorMessage'

/** 操作の結果のトースト。done は済んだこと、alert は状態が変わっていて一覧を取り直したこと */
export type AdminNotice = { message: string; tone: 'done' | 'alert' }

/**
 * 管理画面の1件への操作（停止・削除・キャンセルなど。ADMIN）。同時に1つだけ実行し、済んだらトーストを少しだけ出す。
 * 状態が変わっていた（409）・見つからない（404）ときは onStale で一覧を取り直し、その理由をトーストに出す（AdminCard と同じ扱い）。
 * それ以外の失敗（期限の日付が不正など）は error として操作の近くに出す。
 */
export function useAdminAction<A extends string>(onStale: () => void) {
  const [running, setRunning] = useState<A | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [notice, setNotice] = useState<AdminNotice>()

  useEffect(() => {
    if (!notice) return
    const timer = setTimeout(() => setNotice(undefined), 3000)
    return () => clearTimeout(timer)
  }, [notice])

  /**
   * call を実行し、成功したら done に結果を渡して、返した文をトーストに出す。
   * 操作を終えたか（成功・一覧の取り直し）を返す。false は失敗の文を操作の近くに出したとき（確認を開いたままにする）
   */
  const run = useCallback(
    async <R>(action: A, call: () => Promise<R>, done: (result: R) => string) => {
      setRunning(action)
      setError(null)
      try {
        const result = await call()
        setNotice({ message: done(result), tone: 'done' })
        return true
      } catch (err) {
        if (
          isApiError(err) &&
          (err.code === 'invalid_status_transition' || err.code === 'not_found')
        ) {
          onStale()
          setNotice({ message: err.message, tone: 'alert' })
          return true
        }
        setError(messageOf(err))
        return false
      } finally {
        setRunning(null)
      }
    },
    [onStale],
  )

  /** 選び直したときなど、前の失敗の文を消す */
  const clearError = useCallback(() => setError(null), [])

  return { run, running, error, clearError, notice }
}
