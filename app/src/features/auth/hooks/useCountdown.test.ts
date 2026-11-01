import { act, renderHook } from '@testing-library/react'
import { useCountdown } from './useCountdown'

describe('useCountdown', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-10-16T12:00:00+09:00'))
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  it('期限までの残り秒数を1秒ごとに減らす', () => {
    const deadline = Date.now() + 3000
    const { result } = renderHook(() => useCountdown(deadline))
    expect(result.current).toBe(3)
    act(() => vi.advanceTimersByTime(1000))
    expect(result.current).toBe(2)
  })

  it('期限を過ぎたら 0 で止まる', () => {
    const deadline = Date.now() + 1000
    const { result } = renderHook(() => useCountdown(deadline))
    act(() => vi.advanceTimersByTime(5000))
    expect(result.current).toBe(0)
  })

  it('期限が変わったら数え直す', () => {
    const { result, rerender } = renderHook(({ deadline }) => useCountdown(deadline), {
      initialProps: { deadline: Date.now() },
    })
    expect(result.current).toBe(0)
    rerender({ deadline: Date.now() + 60_000 })
    expect(result.current).toBe(60)
    act(() => vi.advanceTimersByTime(1000))
    expect(result.current).toBe(59)
  })
})
