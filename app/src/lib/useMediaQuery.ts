import { useCallback, useSyncExternalStore } from 'react'

/** PC 幅の上部ナビに切り替える幅（DS-1。CSS の @media (min-width: 1024px) と同じ値） */
export const PC_MEDIA_QUERY = '(min-width: 1024px)'

/**
 * メディアクエリに合っているかを返し、画面幅が変わったら描画し直す。
 * matchMedia がない環境（jsdom のテストなど）では false（スマホの扱い）を返す。
 */
export function useMediaQuery(query: string): boolean {
  const subscribe = useCallback(
    (onChange: () => void) => {
      if (typeof window.matchMedia !== 'function') return () => {}
      const list = window.matchMedia(query)
      list.addEventListener('change', onChange)
      return () => list.removeEventListener('change', onChange)
    },
    [query],
  )
  const getSnapshot = () =>
    typeof window.matchMedia === 'function' && window.matchMedia(query).matches
  return useSyncExternalStore(subscribe, getSnapshot)
}
