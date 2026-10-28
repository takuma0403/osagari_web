package domain

import "time"

// FeeRateBasisPointsMax は手数料率の上限（100% = 10000 ベーシスポイント。PAY-2）。
const FeeRateBasisPointsMax = 10000

// CalcFee は商品価格（円）と手数料率（ベーシスポイント）から、手数料と出品者の売上を求める（PAY-2）。
// 手数料は 価格 × 料率 の1円未満を切り捨てる（ADR 0017）。出品者の売上 = 価格 − 手数料。
func CalcFee(price int64, feeRateBP int) (fee, proceeds int64) {
	fee = price * int64(feeRateBP) / FeeRateBasisPointsMax
	return fee, price - fee
}

// FeeRateSetting は手数料率と、最後に変更した日時・管理者（PAY-2, ADR 0034）。
type FeeRateSetting struct {
	BasisPoints int
	UpdatedAt   time.Time
	// UpdatedBy は最後に変更した管理者のユーザーID。初期値のままなら nil
	UpdatedBy *string
}
