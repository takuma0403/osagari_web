package domain

import "time"

// Transaction は取引（transactions）。購入希望の段階から1件の取引として作る（DEAL-1, DATA）。
// 受け渡しトークンは HandoverToken として別に持つ。
type Transaction struct {
	ID       string
	ItemID   string
	BuyerID  string
	SellerID string
	// Price は購入希望の時点の商品価格（円）
	Price int64
	// PaymentMethod は購入者が購入希望のときに選んだ支払い方法（ADR 0021）
	PaymentMethod PaymentMethod
	Status        TransactionStatus
	// ApprovalDeadline は承認期限（購入希望から48時間。DEAL-1）
	ApprovalDeadline time.Time
	// FeeRateBP・Fee・SellerProceeds は取引成立（承認）の時点で記録する（PAY-2, ADR 0003）。成立前は nil
	FeeRateBP      *int
	Fee            *int64
	SellerProceeds *int64
	// IdentityDisclosedAt は学籍番号を開示した日時（承認時。DEAL-1）
	IdentityDisclosedAt *time.Time
	// SellerAcknowledgedAt は出品者が確認モーダルに同意した日時（DEAL-2）
	SellerAcknowledgedAt *time.Time
	// BuyerCancelAgreedAt・SellerCancelAgreedAt はキャンセルへの同意日時（DEAL-4）
	BuyerCancelAgreedAt  *time.Time
	SellerCancelAgreedAt *time.Time
	// CompletedAt・HandoverMethod は受け渡しが完了した日時と完了方法（DEAL-3, DEAL-5）
	CompletedAt    *time.Time
	HandoverMethod *HandoverMethod
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
