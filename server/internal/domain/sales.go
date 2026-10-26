package domain

import "time"

// SalesTransactionStatuses は売上の明細に出す取引のステータス（取引成立後の取引中・完了。PAY, ADR 0029）。
// キャンセルした取引は返金済みで売上にならないので含めない。
var SalesTransactionStatuses = []TransactionStatus{TransactionStatusInDeal, TransactionStatusCompleted}

// SalesEntry は売上の明細の1行（取引1件。PAY-2）。手数料は取引成立時に記録した値を使う（ADR 0003）。
type SalesEntry struct {
	Transaction *Transaction
	// PaidOutAt は出金した日時。出金前は nil
	PaidOutAt *time.Time
}

// State は取引のステータスと出金の有無から売上の状態を決める。
func (e SalesEntry) State() SalesEntryState {
	switch {
	case e.PaidOutAt != nil:
		return SalesEntryStatePaidOut
	case e.Transaction.Status == TransactionStatusCompleted:
		return SalesEntryStateSettled
	default:
		return SalesEntryStateAwaitingHandover
	}
}

// LastActivityAt は最後に動いた日時（出金・受け渡し完了・取引成立のうち最も新しいもの）。明細の並びに使う。
func (e SalesEntry) LastActivityAt() time.Time {
	switch {
	case e.PaidOutAt != nil:
		return *e.PaidOutAt
	case e.Transaction.CompletedAt != nil:
		return *e.Transaction.CompletedAt
	case e.Transaction.IdentityDisclosedAt != nil:
		return *e.Transaction.IdentityDisclosedAt
	default:
		return e.Transaction.UpdatedAt
	}
}

// Proceeds は出品者の売上（円）。取引成立前で記録がなければ 0。
func (e SalesEntry) Proceeds() int64 {
	if e.Transaction.SellerProceeds == nil {
		return 0
	}
	return *e.Transaction.SellerProceeds
}

// SalesSummary は売上の集計（円）。
type SalesSummary struct {
	// Available は出金できる売上（settled の合計）
	Available int64
	// AwaitingHandover は受け渡し待ちの売上（awaiting_handover の合計）
	AwaitingHandover int64
	// PaidOut は出金済みの売上（paid_out の合計）
	PaidOut int64
}

// SummarizeSales は明細を状態ごとに合計する。金額は記録済みの売上をそのまま足し、計算し直さない（PAY-2）。
func SummarizeSales(entries []SalesEntry) SalesSummary {
	var s SalesSummary
	for _, e := range entries {
		switch e.State() {
		case SalesEntryStateSettled:
			s.Available += e.Proceeds()
		case SalesEntryStateAwaitingHandover:
			s.AwaitingHandover += e.Proceeds()
		case SalesEntryStatePaidOut:
			s.PaidOut += e.Proceeds()
		}
	}
	return s
}
