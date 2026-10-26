import { useCallback, useEffect, useRef, useState } from 'react'
import { isApiError } from '../../../api/client'
import { listTransactions, type TransactionListEntry, type TransactionRole } from '../api'

const unexpectedMessage = '読み込めませんでした。もう一度お試しください'

function messageOf(err: unknown): string {
  return isApiError(err) ? err.message : unexpectedMessage
}

export type DealListState =
  | { status: 'loading' }
  | { status: 'error'; message: string }
  | {
      status: 'ready'
      /** 進行中（全件） */
      active: TransactionListEntry[]
      /** 終了した取引（読み込んだところまで） */
      ended: TransactionListEntry[]
      nextCursor: string | null
      /** 「もっと見る」で続きを読み込み中 */
      loadingMore: boolean
      /** 続きの読み込みに失敗したときの文 */
      moreError: string | null
    }

/** 読み込んだ結果と、それがどの区分・何回目の読み込みのものか */
type Result = { key: string; state: DealListState }

/**
 * 取引一覧（DEAL-1、DealList）の購入か出品の、進行中と終了した取引を並行して読み込む（ADR 0028）。
 * role が変わったら取り直し、前の読み込みは止める。終了した取引の続きは loadMore で nextCursor を使って足す。
 */
export function useDealList(role: TransactionRole) {
  const [reloadCount, setReloadCount] = useState(0)
  const key = `${reloadCount}:${role}`
  const [result, setResult] = useState<Result>()
  const moreController = useRef<AbortController | null>(null)

  // いまの区分の結果がまだなければ読み込み中
  const state: DealListState = result?.key === key ? result.state : { status: 'loading' }

  useEffect(() => {
    const controller = new AbortController()
    moreController.current?.abort()
    Promise.all([
      listTransactions(role, 'active', undefined, controller.signal),
      listTransactions(role, 'ended', undefined, controller.signal),
    ]).then(
      ([active, ended]) =>
        setResult({
          key,
          state: {
            status: 'ready',
            active: active.entries,
            ended: ended.entries,
            nextCursor: ended.nextCursor,
            loadingMore: false,
            moreError: null,
          },
        }),
      (err) => {
        if (!controller.signal.aborted)
          setResult({ key, state: { status: 'error', message: messageOf(err) } })
      },
    )
    return () => controller.abort()
  }, [key, role])

  // 画面を離れたら続きの読み込みも止める
  useEffect(() => () => moreController.current?.abort(), [])

  const cursor = state.status === 'ready' && !state.loadingMore ? state.nextCursor : null

  const loadMore = useCallback(() => {
    if (!cursor) return
    const controller = new AbortController()
    moreController.current = controller
    // 同じ区分の結果にだけ反映する（読み込み中にタブが変わったら捨てる）
    const updateReady = (
      change: (s: Extract<DealListState, { status: 'ready' }>) => DealListState,
    ) =>
      setResult((r) =>
        r?.key === key && r.state.status === 'ready' ? { key, state: change(r.state) } : r,
      )

    updateReady((s) => ({ ...s, loadingMore: true, moreError: null }))
    listTransactions(role, 'ended', cursor, controller.signal).then(
      (page) =>
        updateReady((s) => ({
          ...s,
          ended: [...s.ended, ...page.entries],
          nextCursor: page.nextCursor,
          loadingMore: false,
        })),
      (err) => {
        if (!controller.signal.aborted)
          updateReady((s) => ({ ...s, loadingMore: false, moreError: messageOf(err) }))
      },
    )
  }, [cursor, key, role])

  const reload = useCallback(() => setReloadCount((n) => n + 1), [])

  return { state, loadMore, reload }
}
