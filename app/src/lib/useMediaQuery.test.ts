import { act, renderHook } from '@testing-library/react'
import { useMediaQuery } from './useMediaQuery'

/** matchMedia を偽装する。返した setMatches で幅が変わったことにする */
function mockMatchMedia(initial: boolean) {
  let matches = initial
  const listeners = new Set<() => void>()
  vi.stubGlobal(
    'matchMedia',
    vi.fn((query: string) => ({
      get matches() {
        return matches
      },
      media: query,
      addEventListener: (_: string, fn: () => void) => listeners.add(fn),
      removeEventListener: (_: string, fn: () => void) => listeners.delete(fn),
    })),
  )
  return (next: boolean) => {
    matches = next
    listeners.forEach((fn) => fn())
  }
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('useMediaQuery', () => {
  it('メディアクエリに合っているかを返し、幅が変わったら追う', () => {
    const setMatches = mockMatchMedia(false)
    const { result } = renderHook(() => useMediaQuery('(min-width: 1024px)'))
    expect(result.current).toBe(false)
    act(() => setMatches(true))
    expect(result.current).toBe(true)
  })

  it('matchMedia がない環境では false を返す', () => {
    vi.stubGlobal('matchMedia', undefined)
    const { result } = renderHook(() => useMediaQuery('(min-width: 1024px)'))
    expect(result.current).toBe(false)
  })
})
