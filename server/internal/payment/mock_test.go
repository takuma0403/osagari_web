package payment_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/clock"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/payment"
	"github.com/takuma0403/osagari_web/server/internal/repository"
	"github.com/takuma0403/osagari_web/server/internal/testutil/testdb"
)

var t0 = time.Date(2026, 10, 22, 9, 0, 0, 0, time.UTC)

func ptr[T any](v T) *T { return &v }

// newTransaction は出品者・購入者・商品をそろえ、成立済み（手数料を記録した）取引を保存して返す。
func newTransaction(t *testing.T, ctx context.Context, txs *repository.TransactionRepository, users *repository.UserRepository, items *repository.ItemRepository) *domain.Transaction {
	t.Helper()
	var ids []string
	for _, n := range []string{"26238001", "26238002"} {
		u := &domain.User{
			Email: n + "@edu.cc.saga-u.ac.jp", StudentNumber: n, AdmissionYear: 2026, DepartmentCode: "238",
			ExpiresOn: time.Date(2030, 3, 31, 0, 0, 0, 0, time.UTC), Status: domain.UserStatusActive,
			SellerStatus: domain.SellerStatusNone, PasswordHash: "hash", CodeVerifiedAt: t0, CreatedAt: t0, UpdatedAt: t0,
		}
		if err := users.Create(ctx, u); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, u.ID)
	}
	it := &domain.Item{
		SellerID: ids[0], ShareID: "share0000001", Status: domain.ItemStatusInDeal,
		ItemContent: domain.ItemContent{Title: "線形代数", Price: ptr[int64](1550)}, CreatedAt: t0, UpdatedAt: t0,
	}
	if err := items.Create(ctx, it); err != nil {
		t.Fatal(err)
	}
	fee, proceeds := domain.CalcFee(1550, 500)
	tr := &domain.Transaction{
		ItemID: it.ID, SellerID: ids[0], BuyerID: ids[1], Price: 1550, PaymentMethod: domain.PaymentMethodPayPay,
		Status: domain.TransactionStatusInDeal, ApprovalDeadline: t0.Add(48 * time.Hour),
		FeeRateBP: ptr(500), Fee: &fee, SellerProceeds: &proceeds, CreatedAt: t0, UpdatedAt: t0,
	}
	if err := txs.Create(ctx, tr); err != nil {
		t.Fatal(err)
	}
	return tr
}

// mock の各操作が payments に記録されることを確かめる（JOBS-2, PAY-2）。
func TestMock(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	payments := repository.NewPaymentRepository(gdb)
	tr := newTransaction(t, ctx, repository.NewTransactionRepository(gdb), repository.NewUserRepository(gdb), repository.NewItemRepository(gdb))
	clk := clock.NewFake(t0)
	var gw payment.Gateway = payment.NewMock(payments, clk)

	ops := []struct {
		kind   domain.PaymentKind
		run    func(context.Context, *domain.Transaction) error
		amount int64
		method *domain.PaymentMethod
	}{
		{domain.PaymentKindAuthorize, gw.Authorize, 1550, ptr(domain.PaymentMethodPayPay)},
		{domain.PaymentKindCapture, gw.Capture, 1550, ptr(domain.PaymentMethodPayPay)},
		{domain.PaymentKindRelease, gw.Release, 1550, ptr(domain.PaymentMethodPayPay)},
		{domain.PaymentKindRefund, gw.Refund, 1550, ptr(domain.PaymentMethodPayPay)},
		// 売上の確定は手数料（¥77）を引いた額で、支払い方法は空
		{domain.PaymentKindSettle, gw.Settle, 1473, nil},
		// 出金も売上の額で、支払い方法は空（ADR 0029）
		{domain.PaymentKindPayout, gw.Payout, 1473, nil},
	}
	for _, op := range ops {
		clk.Advance(time.Minute)
		if err := op.run(ctx, tr); err != nil {
			t.Fatalf("%s: %v", op.kind, err)
		}
	}

	got, err := payments.ListByTransaction(ctx, tr.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(ops) {
		t.Fatalf("記録の件数 = %d, want %d", len(got), len(ops))
	}
	for i, op := range ops {
		g := got[i]
		want := t0.Add(time.Duration(i+1) * time.Minute)
		if g.Kind != op.kind || g.Amount != op.amount || g.Status != domain.PaymentStatusSucceeded || g.ExternalID != nil ||
			!g.CreatedAt.Equal(want) || g.TransactionID != tr.ID {
			t.Errorf("%s: got = %+v", op.kind, g)
		}
		if (g.PaymentMethod == nil) != (op.method == nil) || (g.PaymentMethod != nil && *g.PaymentMethod != *op.method) {
			t.Errorf("%s: 支払い方法 = %v, want %v", op.kind, g.PaymentMethod, op.method)
		}
	}
}

func TestMock_成立前の売上の確定(t *testing.T) {
	gw := payment.NewMock(nil, clock.NewFake(t0))
	tr := &domain.Transaction{ID: "tx", Price: 1550, PaymentMethod: domain.PaymentMethodCard, Status: domain.TransactionStatusRequested}
	if err := gw.Settle(context.Background(), tr); !errors.Is(err, payment.ErrFeeNotRecorded) {
		t.Errorf("Settle: err = %v, want ErrFeeNotRecorded", err)
	}
	if err := gw.Payout(context.Background(), tr); !errors.Is(err, payment.ErrFeeNotRecorded) {
		t.Errorf("Payout: err = %v, want ErrFeeNotRecorded", err)
	}
}

// 出金は取引ごとに1行だけ。2回目は一意制約で ErrDuplicate になる（ADR 0029）。
func TestMock_二重の出金(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	payments := repository.NewPaymentRepository(gdb)
	tr := newTransaction(t, ctx, repository.NewTransactionRepository(gdb), repository.NewUserRepository(gdb), repository.NewItemRepository(gdb))
	gw := payment.NewMock(payments, clock.NewFake(t0))

	if err := gw.Payout(ctx, tr); err != nil {
		t.Fatal(err)
	}
	if err := gw.Payout(ctx, tr); !errors.Is(err, repository.ErrDuplicate) {
		t.Errorf("2回目の出金: err = %v, want ErrDuplicate", err)
	}
	// 出金以外の種別は同じ取引に何行でも記録できる
	for range 2 {
		if err := gw.Settle(ctx, tr); err != nil {
			t.Fatal(err)
		}
	}
}

// 呼び出し側のトランザクションがロールバックされたら、記録も残らない。
func TestMock_トランザクション(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	payments := repository.NewPaymentRepository(gdb)
	tr := newTransaction(t, ctx, repository.NewTransactionRepository(gdb), repository.NewUserRepository(gdb), repository.NewItemRepository(gdb))
	gw := payment.NewMock(payments, clock.NewFake(t0))

	errRollback := errors.New("rollback")
	err := repository.NewTransactor(gdb).Do(ctx, func(ctx context.Context) error {
		if err := gw.Capture(ctx, tr); err != nil {
			return err
		}
		return errRollback
	})
	if !errors.Is(err, errRollback) {
		t.Fatalf("err = %v", err)
	}
	got, err := payments.ListByTransaction(ctx, tr.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("ロールバックしたのに記録が残っている: %+v", got)
	}
}
