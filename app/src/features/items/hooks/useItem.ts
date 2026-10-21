import { useCallback, useEffect, useState } from 'react'
import { isApiError } from '../../../api/client'
import { getPublicItem, type PublicItem } from '../api'

const unexpectedMessage = '読み込めませんでした。もう一度お試しください'

export type ItemState =
  | { status: 'loading' }
  /** 下書き・取り下げ・停止、または出品者が使えない商品（ADR 0018） */
  | { status: 'notFound' }
  | { status: 'error'; message: string }
  | { status: 'ready'; item: PublicItem }

/** 読み込んだ結果と、それがどの商品・何回目の読み込みのものか */
type Result = { key: string; state: ItemState }

/** 商品ページ（/items/:shareId）の商品を読み込む（BROWSE）。shareId が変わったら取り直し、前の読み込みは止める */
export function useItem(shareId: string) {
  const [reloadCount, setReloadCount] = useState(0)
  const key = `${reloadCount}/${shareId}`
  const [result, setResult] = useState<Result>()

  // いまの商品の結果がまだなければ読み込み中
  const state: ItemState = result?.key === key ? result.state : { status: 'loading' }

  useEffect(() => {
    const controller = new AbortController()
    getPublicItem(shareId, controller.signal).then(
      (item) => setResult({ key, state: { status: 'ready', item } }),
      (err: unknown) => {
        if (controller.signal.aborted) return
        const state: ItemState =
          isApiError(err) && err.code === 'not_found'
            ? { status: 'notFound' }
            : { status: 'error', message: isApiError(err) ? err.message : unexpectedMessage }
        setResult({ key, state })
      },
    )
    return () => controller.abort()
  }, [key, shareId])

  const reload = useCallback(() => setReloadCount((n) => n + 1), [])

  return { state, reload }
}
