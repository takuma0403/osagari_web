import { useCallback, useEffect, useState } from 'react'
import { isApiError } from '../../../api/client'
import { getTransaction, type Transaction } from '../api'

const unexpectedMessage = '読み込めませんでした。もう一度お試しください'

export type TransactionState =
  | { status: 'loading' }
  /** 当事者でない、または存在しない取引 */
  | { status: 'notFound' }
  | { status: 'error'; message: string }
  | { status: 'ready'; transaction: Transaction }

/** 読み込んだ結果と、それがどの取引のものか */
type Result = { dealId: string; state: TransactionState }

/**
 * 取引を読み込む（DEAL-1〜3）。dealId が変わったら取り直し、前の読み込みは止める。
 * 取り直している間・取り直しに失敗したときは、同じ取引の前の結果を出したままにする（受け渡しQRの表示中の定期的な取り直しで画面を崩さないため）。
 */
export function useTransaction(dealId: string) {
  const [reloadCount, setReloadCount] = useState(0)
  const [result, setResult] = useState<Result>()

  const state: TransactionState = result?.dealId === dealId ? result.state : { status: 'loading' }

  useEffect(() => {
    const controller = new AbortController()
    getTransaction(dealId, controller.signal).then(
      (transaction) => setResult({ dealId, state: { status: 'ready', transaction } }),
      (err: unknown) => {
        if (controller.signal.aborted) return
        const state: TransactionState =
          isApiError(err) && err.code === 'not_found'
            ? { status: 'notFound' }
            : { status: 'error', message: isApiError(err) ? err.message : unexpectedMessage }
        setResult((prev) =>
          prev?.dealId === dealId && prev.state.status === 'ready' ? prev : { dealId, state },
        )
      },
    )
    return () => controller.abort()
  }, [dealId, reloadCount])

  const reload = useCallback(() => setReloadCount((n) => n + 1), [])

  return { state, reload }
}
