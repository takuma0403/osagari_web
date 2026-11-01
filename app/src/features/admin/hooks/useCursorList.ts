import { useCallback, useEffect, useState } from 'react'
import { messageOf } from '../errorMessage'

type Page<T> = { entries: T[]; nextCursor: string | null }

type FetchPage<P> = (cursor: string | undefined, signal: AbortSignal) => Promise<P>

/** 読み込んだ一覧。どの条件（fetchPage）と読み直しの回数で読んだかを持ち、条件が変わったら古い値を出さない */
type Loaded<T, P> = {
  fetchPage: FetchPage<P>
  reloadKey: number
  head?: P
  entries?: T[]
  nextCursor: string | null
  loadError: string | null
  moreError: string | null
}

/**
 * 管理画面の一覧（ユーザー・出品・取引）を先頭から読み込み、「もっと見る」で nextCursor の続きを足す（ADMIN）。
 * fetchPage が変わったら（検索条件を変えたら）先頭から読み直す。head は先頭の応答（件数など一覧の外の値を読む）。
 */
export function useCursorList<T extends { id: string }, P extends Page<T>>(
  fetchPage: FetchPage<P>,
) {
  const [reloadKey, setReloadKey] = useState(0)
  const [loaded, setLoaded] = useState<Loaded<T, P>>()
  const [loadingMore, setLoadingMore] = useState(false)

  useEffect(() => {
    const controller = new AbortController()
    const base = { fetchPage, reloadKey, moreError: null }
    fetchPage(undefined, controller.signal)
      .then((page) =>
        setLoaded({
          ...base,
          head: page,
          entries: page.entries,
          nextCursor: page.nextCursor,
          loadError: null,
        }),
      )
      .catch((err: unknown) => {
        if (!controller.signal.aborted)
          setLoaded({ ...base, nextCursor: null, loadError: messageOf(err) })
      })
    return () => controller.abort()
  }, [fetchPage, reloadKey])

  // 条件を変えた・読み直した直後は、新しい応答が来るまで読み込み中にする
  const current =
    loaded && loaded.fetchPage === fetchPage && loaded.reloadKey === reloadKey ? loaded : undefined

  const reload = useCallback(() => setReloadKey((key) => key + 1), [])

  const loadMore = useCallback(async () => {
    const cursor = current?.nextCursor
    if (!cursor) return
    setLoadingMore(true)
    setLoaded((prev) => prev && { ...prev, moreError: null })
    try {
      const page = await fetchPage(cursor, new AbortController().signal)
      setLoaded((prev) =>
        // 読み込みのあいだに条件を変えていたら、前の条件の続きは足さない
        prev?.fetchPage !== fetchPage
          ? prev
          : {
              ...prev,
              // 読み込みのあいだに動いた行が前のページと重なっても、同じ行を2回出さない
              entries: [
                ...(prev.entries ?? []),
                ...page.entries.filter((e) => !prev.entries?.some((p) => p.id === e.id)),
              ],
              nextCursor: page.nextCursor,
            },
      )
    } catch (err) {
      setLoaded((prev) => prev && { ...prev, moreError: messageOf(err) })
    } finally {
      setLoadingMore(false)
    }
  }, [current?.nextCursor, fetchPage])

  /** 操作が済んだ行を、応答の値に置き換える */
  const replace = useCallback((entry: T) => {
    setLoaded(
      (prev) =>
        prev && { ...prev, entries: prev.entries?.map((e) => (e.id === entry.id ? entry : e)) },
    )
  }, [])

  return {
    head: current?.head,
    entries: current?.entries,
    nextCursor: current?.nextCursor ?? null,
    loadError: current?.loadError ?? null,
    reload,
    loadMore,
    loadingMore,
    moreError: current?.moreError ?? null,
    replace,
  }
}
