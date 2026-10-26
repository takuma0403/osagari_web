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

// TransactionRoles は TransactionRole のすべての値。
var TransactionRoles = []TransactionRole{TransactionRoleBuyer, TransactionRoleSeller}

func (v TransactionRole) Valid() bool { return enumValid(v, TransactionRoles) }

// 取引一覧の1回に返す件数（終了した取引。ADR 0028）。
const (
	TransactionListDefaultLimit = 20
	TransactionListMaxLimit     = 50
)

// TransactionListStatuses は取引一覧で role・phase の区分に出す取引のステータス（ADR 0028）。
// 出品側の進行中の購入希望（requested）は商品ごとにまとめて別に数えるため含めない。出品側の却下・期限切れは出さない。
func TransactionListStatuses(role TransactionRole, phase TransactionPhase) []TransactionStatus {
	switch {
	case role == TransactionRoleBuyer && phase == TransactionPhaseActive:
		return []TransactionStatus{TransactionStatusRequested, TransactionStatusInDeal}
	case role == TransactionRoleBuyer && phase == TransactionPhaseEnded:
		return []TransactionStatus{TransactionStatusRejected, TransactionStatusExpired, TransactionStatusCancelled, TransactionStatusCompleted}
	case role == TransactionRoleSeller && phase == TransactionPhaseActive:
		return []TransactionStatus{TransactionStatusInDeal}
	case role == TransactionRoleSeller && phase == TransactionPhaseEnded:
		return []TransactionStatus{TransactionStatusCancelled, TransactionStatusCompleted}
	}
	return nil
}

// TransactionCursor は取引一覧の続きの位置。最後に動いた順（UpdatedAt の新しい順、同じなら ID の大きい順）で、この取引より後ろを返す。
type TransactionCursor struct {
	UpdatedAt time.Time
	ID        string
}

// TransactionRequestGroup は出品者の商品ごとの承認待ちの購入希望のまとまり（取引一覧の出品側。ADR 0028）。
type TransactionRequestGroup struct {
	ItemID string
	// Count は承認待ちの購入希望の件数
	Count int
	// LatestRequestedAt は最新の購入希望を出した日時
	LatestRequestedAt time.Time
}
