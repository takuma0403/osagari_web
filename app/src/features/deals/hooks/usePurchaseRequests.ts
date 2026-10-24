import { useCallback, useEffect, useState } from 'react'
import { isApiError } from '../../../api/client'
import { getOwnItem, type OwnItem } from '../../sell/api'
import { listPurchaseRequests, type PurchaseRequest } from '../api'

const unexpectedMessage = '読み込めませんでした。もう一度お試しください'

export type PurchaseRequestsState =
  | { status: 'loading' }
  /** ほかの人の商品、または存在しない商品 */
  | { status: 'notFound' }
  | { status: 'error'; message: string }
  | { status: 'ready'; item: OwnItem; requests: PurchaseRequest[] }

/** 読み込んだ結果と、それがどの商品・何回目の読み込みのものか */
type Result = { itemId: string; reloadCount: number; state: PurchaseRequestsState }

/**
 * 自分の商品と、届いている承認待ちの購入希望を読み込む（DEAL-1、DealRequests）。
 * itemId が変わったら取り直し、前の読み込みは止める。承認・却下の後は reload で取り直す。
 */
export function usePurchaseRequests(itemId: string) {
  const [reloadCount, setReloadCount] = useState(0)
  const [result, setResult] = useState<Result>()

  // いまの商品の結果がまだなければ読み込み中。取り直している間は、同じ商品の前の一覧を出したままにする
  const state: PurchaseRequestsState =
    result?.itemId === itemId &&
    (result.reloadCount === reloadCount || result.state.status === 'ready')
      ? result.state
      : { status: 'loading' }

  useEffect(() => {
    const controller = new AbortController()
    Promise.all([
      getOwnItem(itemId, controller.signal),
      listPurchaseRequests(itemId, controller.signal),
    ]).then(
      ([item, list]) =>
        setResult({
          itemId,
          reloadCount,
          state: { status: 'ready', item, requests: list.requests },
        }),
      (err: unknown) => {
        if (controller.signal.aborted) return
        const state: PurchaseRequestsState =
          isApiError(err) && err.code === 'not_found'
            ? { status: 'notFound' }
            : { status: 'error', message: isApiError(err) ? err.message : unexpectedMessage }
        setResult({ itemId, reloadCount, state })
      },
    )
    return () => controller.abort()
  }, [itemId, reloadCount])

  const reload = useCallback(() => setReloadCount((n) => n + 1), [])

  return { state, reload }
}
