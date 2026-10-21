import { useCallback, useEffect, useRef, useState } from 'react'
import { isApiError } from '../../../api/client'
import { listItems, type ItemSummary } from '../api'
import {
  parseSearchConditions,
  toSearchParams,
  type ItemSearchConditions,
} from '../searchConditions'

const unexpectedMessage = '読み込めませんでした。もう一度お試しください'

function messageOf(err: unknown): string {
  return isApiError(err) ? err.message : unexpectedMessage
}

export type ItemListState =
  | { status: 'loading' }
  | { status: 'error'; message: string }
  | {
      status: 'ready'
      items: ItemSummary[]
      /** 条件に合う商品の総数（読み込んだ件数ではない） */
      total: number
      nextCursor: string | null
      /** 「もっと見る」で続きを読み込み中 */
      loadingMore: boolean
      /** 続きの読み込みに失敗したときの文 */
      moreError: string | null
    }

/** 読み込んだ結果と、それがどの条件・何回目の読み込みのものか */
type Result = { key: string; state: ItemListState }

/**
 * 公開中の商品を条件で新着順に読み込む（BROWSE）。条件が変わったら1ページ目から取り直し、
 * 前の読み込みは止める。続きは loadMore で nextCursor を使って足す。
 */
export function useItemList(conditions: ItemSearchConditions) {
  // 条件はオブジェクトなので、同じ条件かどうかはクエリ文字列で比べる
  const query = toSearchParams(conditions).toString()
  const [reloadCount, setReloadCount] = useState(0)
  const key = `${reloadCount}?${query}`
  const [result, setResult] = useState<Result>()
  const moreController = useRef<AbortController | null>(null)

  // いまの条件の結果がまだなければ読み込み中
  const state: ItemListState = result?.key === key ? result.state : { status: 'loading' }

  useEffect(() => {
    const controller = new AbortController()
    moreController.current?.abort()
    listItems(parseSearchConditions(new URLSearchParams(query)), undefined, controller.signal).then(
      (page) =>
        setResult({
          key,
          state: {
            status: 'ready',
            items: page.items,
            total: page.total,
            nextCursor: page.nextCursor,
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
  }, [key, query])

  // 画面を離れたら続きの読み込みも止める
  useEffect(() => () => moreController.current?.abort(), [])

  const cursor = state.status === 'ready' && !state.loadingMore ? state.nextCursor : null

  const loadMore = useCallback(() => {
    if (!cursor) return
    const controller = new AbortController()
    moreController.current = controller
    // 同じ条件の結果にだけ反映する（読み込み中に条件が変わったら捨てる）
    const updateReady = (
      change: (s: Extract<ItemListState, { status: 'ready' }>) => ItemListState,
    ) =>
      setResult((r) =>
        r?.key === key && r.state.status === 'ready' ? { key, state: change(r.state) } : r,
      )

    updateReady((s) => ({ ...s, loadingMore: true, moreError: null }))
    listItems(parseSearchConditions(new URLSearchParams(query)), cursor, controller.signal).then(
      (page) =>
        updateReady((s) => ({
          ...s,
          items: [...s.items, ...page.items],
          total: page.total,
          nextCursor: page.nextCursor,
          loadingMore: false,
        })),
      (err) => {
        if (!controller.signal.aborted)
          updateReady((s) => ({ ...s, loadingMore: false, moreError: messageOf(err) }))
      },
    )
  }, [cursor, key, query])

  const reload = useCallback(() => setReloadCount((n) => n + 1), [])

  return { state, loadMore, reload }
}
