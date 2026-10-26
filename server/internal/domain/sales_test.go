package domain_test

import (
	"testing"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/domain"
)

func TestSalesEntry(t *testing.T) {
	approved := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	completed := approved.Add(24 * time.Hour)
	paidOut := completed.Add(24 * time.Hour)
	tests := []struct {
		name      string
		status    domain.TransactionStatus
		completed *time.Time
		paidOut   *time.Time
		want      domain.SalesEntryState
		wantAt    time.Time
	}{
		{"取引中は受け渡し待ち", domain.TransactionStatusInDeal, nil, nil, domain.SalesEntryStateAwaitingHandover, approved},
		{"完了して未出金は確定", domain.TransactionStatusCompleted, &completed, nil, domain.SalesEntryStateSettled, completed},
		{"出金済み", domain.TransactionStatusCompleted, &completed, &paidOut, domain.SalesEntryStatePaidOut, paidOut},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := domain.SalesEntry{
				Transaction: &domain.Transaction{Status: tt.status, IdentityDisclosedAt: &approved, CompletedAt: tt.completed},
				PaidOutAt:   tt.paidOut,
			}
			if got := e.State(); got != tt.want {
				t.Errorf("State() = %q, want %q", got, tt.want)
			}
			if got := e.LastActivityAt(); !got.Equal(tt.wantAt) {
				t.Errorf("LastActivityAt() = %v, want %v", got, tt.wantAt)
			}
		})
	}
}

func TestSummarizeSales(t *testing.T) {
	at := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	entry := func(status domain.TransactionStatus, price int64, bp int, paidOut bool) domain.SalesEntry {
		fee, proceeds := domain.CalcFee(price, bp)
		e := domain.SalesEntry{Transaction: &domain.Transaction{
			Status: status, Price: price, FeeRateBP: &bp, Fee: &fee, SellerProceeds: &proceeds, IdentityDisclosedAt: &at,
		}}
		if paidOut {
			e.PaidOutAt = &at
		}
		return e
	}
	got := domain.SummarizeSales([]domain.SalesEntry{
		entry(domain.TransactionStatusCompleted, 1550, 500, false),  // 手数料 77（切り捨て）、売上 1473
		entry(domain.TransactionStatusCompleted, 2700, 1000, false), // 料率を変えた後に成立した取引。売上 2430
		entry(domain.TransactionStatusInDeal, 800, 1000, false),     // 売上 720
		entry(domain.TransactionStatusCompleted, 3500, 1000, true),  // 売上 3150
	})
	want := domain.SalesSummary{Available: 1473 + 2430, AwaitingHandover: 720, PaidOut: 3150}
	if got != want {
		t.Errorf("SummarizeSales() = %+v, want %+v", got, want)
	}
	if got := domain.SummarizeSales(nil); got != (domain.SalesSummary{}) {
		t.Errorf("空の明細の集計 = %+v, want ゼロ", got)
	}
}
