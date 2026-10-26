package payment

import (
	"context"

	"github.com/takuma0403/osagari_web/server/internal/clock"
	"github.com/takuma0403/osagari_web/server/internal/domain"
)

// Recorder は決済の記録を保存する（repository.PaymentRepository）。
type Recorder interface {
	Create(ctx context.Context, p *domain.Payment) error
}

// Mock はフェーズ1の Gateway。各操作を成功したことにして payments に1行記録するだけで、外部IDは空のまま（JOBS-2）。
type Mock struct {
	recorder Recorder
	clock    clock.Clock
}

func NewMock(recorder Recorder, clk clock.Clock) *Mock {
	return &Mock{recorder: recorder, clock: clk}
}

var _ Gateway = (*Mock)(nil)

func (m *Mock) Authorize(ctx context.Context, tr *domain.Transaction) error {
	return m.record(ctx, tr, domain.PaymentKindAuthorize, tr.Price, &tr.PaymentMethod)
}

func (m *Mock) Capture(ctx context.Context, tr *domain.Transaction) error {
	return m.record(ctx, tr, domain.PaymentKindCapture, tr.Price, &tr.PaymentMethod)
}

func (m *Mock) Release(ctx context.Context, tr *domain.Transaction) error {
	return m.record(ctx, tr, domain.PaymentKindRelease, tr.Price, &tr.PaymentMethod)
}

func (m *Mock) Refund(ctx context.Context, tr *domain.Transaction) error {
	return m.record(ctx, tr, domain.PaymentKindRefund, tr.Price, &tr.PaymentMethod)
}

// Settle は出品者の売上（取引成立時に記録した SellerProceeds）を記録する。支払い方法は空にする（ADR 0021）。
func (m *Mock) Settle(ctx context.Context, tr *domain.Transaction) error {
	if tr.SellerProceeds == nil {
		return ErrFeeNotRecorded
	}
	return m.record(ctx, tr, domain.PaymentKindSettle, *tr.SellerProceeds, nil)
}

// Payout は出品者の売上（SellerProceeds）を出金として記録する。支払い方法は空にする（ADR 0021）。
// 取引ごとに1行だけで、すでに出金した取引ならリポジトリが一意制約違反を返す（ADR 0029）。
func (m *Mock) Payout(ctx context.Context, tr *domain.Transaction) error {
	if tr.SellerProceeds == nil {
		return ErrFeeNotRecorded
	}
	return m.record(ctx, tr, domain.PaymentKindPayout, *tr.SellerProceeds, nil)
}

func (m *Mock) record(ctx context.Context, tr *domain.Transaction, kind domain.PaymentKind, amount int64, method *domain.PaymentMethod) error {
	now := m.clock.Now()
	return m.recorder.Create(ctx, &domain.Payment{
		TransactionID: tr.ID,
		Kind:          kind,
		PaymentMethod: method,
		Amount:        amount,
		Status:        domain.PaymentStatusSucceeded,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
}
