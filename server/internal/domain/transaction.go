package domain

import "time"

// ApprovalTTL は購入希望の承認期限（購入希望から48時間。DEAL-1, ADR 0022）。
const ApprovalTTL = 48 * time.Hour

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

// RoleOf は userID がこの取引の購入者か出品者かを返す。どちらでもなければ ok が false。
func (t Transaction) RoleOf(userID string) (role TransactionRole, ok bool) {
	switch userID {
	case t.BuyerID:
		return TransactionRoleBuyer, true
	case t.SellerID:
		return TransactionRoleSeller, true
	}
	return "", false
}

// CancelAgreedAt は role の当事者がキャンセルに同意（申請）した日時を返す。同意していなければ nil（DEAL-4）。
func (t Transaction) CancelAgreedAt(role TransactionRole) *time.Time {
	if role == TransactionRoleSeller {
		return t.SellerCancelAgreedAt
	}
	return t.BuyerCancelAgreedAt
}

// CounterpartID は userID から見た取引の相手のユーザーIDを返す。
func (t Transaction) CounterpartID(userID string) string {
	if userID == t.BuyerID {
		return t.SellerID
	}
	return t.BuyerID
}

// TransactionRole はログイン中の本人がその取引の購入者か出品者か。
type TransactionRole string

const (
	TransactionRoleBuyer  TransactionRole = "buyer"
	TransactionRoleSeller TransactionRole = "seller"
)
