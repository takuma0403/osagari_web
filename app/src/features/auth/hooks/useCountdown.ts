import { useEffect, useState } from 'react'

/**
 * deadline（ミリ秒の時刻）までの残り秒数を返し、1秒ごとに更新する。期限を過ぎたら 0 で止まる。
 * 認証コードの有効期限と再送までの待ち時間に使う。
 */
export function useCountdown(deadline: number): number {
  const [now, setNow] = useState(Date.now)

  useEffect(() => {
    if (Date.now() >= deadline) return
    const timer = setInterval(() => {
      const current = Date.now()
      setNow(current)
      if (current >= deadline) clearInterval(timer)
    }, 1000)
    return () => clearInterval(timer)
  }, [deadline])

  return Math.max(0, (deadline - now) / 1000)
}
