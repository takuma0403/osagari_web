import { useCallback, useEffect, useState } from 'react'
import { isApiError } from '../../../api/client'
import { getSales, requestPayout, type Sales } from '../api'

const loadErrorMessage = '読み込めませんでした。もう一度お試しください'
const payoutErrorMessage = '出金を申請できませんでした。もう一度お試しください'

export type SalesState =
  { status: 'loading' } | { status: 'error'; message: string } | { status: 'ready'; sales: Sales }

/** 読み込んだ結果と、それが何回目の読み込みのものか */
type Result = { key: number; state: SalesState }

/**
 * 売上管理（PAY-5、Sales）の売上の明細と集計の読み込みと、出金の申請（モック）。
 * 金額はサーバーが取引成立時に記録した値をそのまま使い、ここでは計算しない（PAY-2）。
 */
export function useSales() {
  const [reloadCount, setReloadCount] = useState(0)
  const [result, setResult] = useState<Result>()
  const [paying, setPaying] = useState(false)
  const [payoutError, setPayoutError] = useState<string | null>(null)

  // いまの読み込みの結果がまだなければ読み込み中
  const state: SalesState = result?.key === reloadCount ? result.state : { status: 'loading' }

  useEffect(() => {
    const controller = new AbortController()
    getSales(controller.signal).then(
      (sales) => setResult({ key: reloadCount, state: { status: 'ready', sales } }),
      (err: unknown) => {
        if (!controller.signal.aborted)
          setResult({
            key: reloadCount,
            state: { status: 'error', message: isApiError(err) ? err.message : loadErrorMessage },
          })
      },
    )
    return () => controller.abort()
  }, [reloadCount])

  const reload = useCallback(() => setReloadCount((n) => n + 1), [])

  /** 出金を申請する。出金できたら true（応答の出金後の売上で表示を置き換える） */
  const payout = useCallback(async () => {
    setPaying(true)
    setPayoutError(null)
    try {
      const sales = await requestPayout()
      setResult({ key: reloadCount, state: { status: 'ready', sales } })
      return true
    } catch (err) {
      setPayoutError(isApiError(err) ? err.message : payoutErrorMessage)
      // ほかの端末で出金済みなど（409）のときは、最新の売上を取り直す
      if (isApiError(err) && err.code === 'invalid_status_transition') reload()
      return false
    } finally {
      setPaying(false)
    }
  }, [reloadCount, reload])

  const clearPayoutError = useCallback(() => setPayoutError(null), [])

  return { state, reload, payout, paying, payoutError, clearPayoutError }
}
