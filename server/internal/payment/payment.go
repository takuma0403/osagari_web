// Package payment は決済の窓口（Gateway）を置く（JOBS-2, PAY）。
// フェーズ1は Mock を差し込み、payments に記録するだけで外部の決済APIは呼ばない。
// 本番では同じ窓口に Stripe Connect や PAY.JP の実装を差し替える。
package payment

import (
	"context"
	"errors"

	"github.com/takuma0403/osagari_web/server/internal/domain"
)

// Gateway は取引ごとの決済の操作（JOBS-2）。service はこのインターフェース越しに使う。
// ctx にトランザクション（repository.Transactor）があれば、記録は同じトランザクションで行う。
type Gateway interface {
	// Authorize は購入希望のときに代金の枠を確保する
	Authorize(ctx context.Context, tr *domain.Transaction) error
	// Capture は出品者の承認のときに代金を預かる
	Capture(ctx context.Context, tr *domain.Transaction) error
	// Release は却下・期限切れのときに枠を解放する
	Release(ctx context.Context, tr *domain.Transaction) error
	// Refund は双方同意のキャンセルのときに返金する（DEAL-4）
	Refund(ctx context.Context, tr *domain.Transaction) error
	// Settle は受け渡し完了のときに、手数料を引いた金額を出品者の売上として確定する（DEAL-3, PAY-2）
	Settle(ctx context.Context, tr *domain.Transaction) error
	// Payout は売上画面の出金申請のときに、確定した出品者の売上を出金する（PAY, ADR 0029）
	Payout(ctx context.Context, tr *domain.Transaction) error
}

// ErrFeeNotRecorded は売上を確定・出金する取引に手数料が記録されていない（取引が成立していない）ときに返す。
var ErrFeeNotRecorded = errors.New("payment: 取引に手数料が記録されていません")
