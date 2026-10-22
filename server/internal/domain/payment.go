package domain

import "time"

// Payment は決済・出金の記録（payments。PAY, JOBS-2, DATA）。出金も取引ごとに1行で記録する（ADR 0021）。
type Payment struct {
	ID            string
	TransactionID string
	Kind          PaymentKind
	// PaymentMethod は支払い方法。売上の確定（settle）と出金（payout）では nil
	PaymentMethod *PaymentMethod
	// Amount は金額（円）
	Amount int64
	Status PaymentStatus
	// ExternalID は本番の決済サービスのID。mock では nil
	ExternalID *string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
