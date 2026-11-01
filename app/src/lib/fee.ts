/** 手数料率の分母（100% = 10000 ベーシスポイント。PAY-2） */
const BASIS_POINTS = 10000

/**
 * 商品価格（円）と手数料率（ベーシスポイント）から、手数料と出品者の受け取り額を求める（PAY-2）。
 * 手数料は 価格 × 料率 の1円未満を切り捨てる（ADR 0017。server の domain.CalcFee と同じ計算）。
 * フロントで金額を計算するのは出品画面の見込み表示だけ（app/CLAUDE.md）。取引の金額はサーバーの値を表示する。
 */
export function estimateProceeds(price: number, feeRateBasisPoints: number) {
  const fee = Math.floor((price * feeRateBasisPoints) / BASIS_POINTS)
  return { fee, proceeds: price - fee }
}

/** 手数料率（ベーシスポイント）を `5%`・`5.25%` の形にする（小数点以下は2桁まで。末尾の0は省く） */
export function formatFeeRate(feeRateBasisPoints: number): string {
  return `${feeRateBasisPoints / 100}%`
}
