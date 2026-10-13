const yen = new Intl.NumberFormat('ja-JP')

/**
 * 金額（円の整数）を `¥1,500` の形にする（PAY-2）。表示は Outfit フォント（--font-num）で行う。
 * 金額の計算はサーバーの値を使い、ここでは表示だけを担う。
 */
export function formatPrice(amount: number): string {
  return `¥${yen.format(amount)}`
}
