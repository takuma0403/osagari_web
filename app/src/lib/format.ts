const yen = new Intl.NumberFormat('ja-JP')

/**
 * 金額（円の整数）を `¥1,500` の形にする（PAY-2）。表示は Outfit フォント（--font-num）で行う。
 * 金額の計算はサーバーの値を使い、ここでは表示だけを担う。
 */
export function formatPrice(amount: number): string {
  return `¥${yen.format(amount)}`
}

/**
 * 日付（`2030-03-31` の形の文字列）を `2030年3月31日` の形にする。
 * 時刻を持たない日付なので、タイムゾーンで日がずれないよう文字列のまま分解する。
 */
export function formatDate(isoDate: string): string {
  const [year, month, day] = isoDate.split('-').map(Number)
  return `${year}年${month}月${day}日`
}

/** 残り秒数を `9:41` の形（分:秒）にする。認証コードの有効期限や再送までの待ち時間に使う */
export function formatCountdown(seconds: number): string {
  const s = Math.max(0, Math.ceil(seconds))
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`
}
