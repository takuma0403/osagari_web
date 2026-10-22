package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/repository"
	"github.com/takuma0403/osagari_web/server/internal/testutil/testdb"
)

// dealFixture は取引のテストに使う出品者・購入者2人・公開中の商品。
type dealFixture struct {
	seller, buyer, buyer2 *domain.User
	item                  *domain.Item
}

func newDealFixture(t *testing.T, gdb *gorm.DB) dealFixture {
	t.Helper()
	ctx := context.Background()
	users := repository.NewUserRepository(gdb)
	f := dealFixture{
		seller: newUser("", email, "26238001"),
		buyer:  newUser("", "26238002@edu.cc.saga-u.ac.jp", "26238002"),
		buyer2: newUser("", "26238003@edu.cc.saga-u.ac.jp", "26238003"),
	}
	for _, u := range []*domain.User{f.seller, f.buyer, f.buyer2} {
		if err := users.Create(ctx, u); err != nil {
			t.Fatal(err)
		}
	}
	f.item = &domain.Item{
		SellerID: f.seller.ID, ShareID: "share0000001", Status: domain.ItemStatusPublished,
		ItemContent: domain.ItemContent{Title: "線形代数", Price: ptr[int64](1550)},
		PublishedAt: &t0, CreatedAt: t0, UpdatedAt: t0,
	}
	if err := repository.NewItemRepository(gdb).Create(ctx, f.item); err != nil {
		t.Fatal(err)
	}
	return f
}

// newRequest は購入希望（requested）の取引を作る。
func (f dealFixture) newRequest(buyer *domain.User) *domain.Transaction {
	return &domain.Transaction{
		ItemID: f.item.ID, BuyerID: buyer.ID, SellerID: f.seller.ID, Price: 1550,
		PaymentMethod: domain.PaymentMethodCard, Status: domain.TransactionStatusRequested,
		ApprovalDeadline: t0.Add(48 * time.Hour), CreatedAt: t0, UpdatedAt: t0,
	}
}

func TestTransactionRepository(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	f := newDealFixture(t, gdb)
	txs := repository.NewTransactionRepository(gdb)

	// 購入希望：成立前は手数料・各日時が空
	req := f.newRequest(f.buyer)
	if err := txs.Create(ctx, req); err != nil {
		t.Fatal(err)
	}
	if req.ID == "" {
		t.Fatal("ID が採番されていない")
	}
	got, err := txs.FindByID(ctx, req.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ItemID != f.item.ID || got.BuyerID != f.buyer.ID || got.SellerID != f.seller.ID || got.Price != 1550 ||
		got.PaymentMethod != domain.PaymentMethodCard || got.Status != domain.TransactionStatusRequested ||
		!got.ApprovalDeadline.Equal(t0.Add(48*time.Hour)) || !got.CreatedAt.Equal(t0) {
		t.Errorf("購入希望: got = %+v", got)
	}
	if got.FeeRateBP != nil || got.Fee != nil || got.SellerProceeds != nil || got.IdentityDisclosedAt != nil ||
		got.SellerAcknowledgedAt != nil || got.BuyerCancelAgreedAt != nil || got.SellerCancelAgreedAt != nil ||
		got.CompletedAt != nil || got.HandoverMethod != nil {
		t.Errorf("購入希望の成立後の項目が空でない: got = %+v", got)
	}

	// 完了した取引：すべての項目が読み戻せる
	fee, proceeds := domain.CalcFee(1550, 500)
	t1 := t0.Add(time.Hour)
	done := f.newRequest(f.buyer2)
	done.PaymentMethod = domain.PaymentMethodPayPay
	done.Status = domain.TransactionStatusCompleted
	done.FeeRateBP, done.Fee, done.SellerProceeds = ptr(500), &fee, &proceeds
	done.IdentityDisclosedAt, done.SellerAcknowledgedAt, done.CompletedAt = &t0, &t1, &t1
	done.BuyerCancelAgreedAt = &t1
	done.HandoverMethod = ptr(domain.HandoverMethodQR)
	if err := txs.Create(ctx, done); err != nil {
		t.Fatal(err)
	}
	got, err = txs.FindByID(ctx, done.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.PaymentMethod != domain.PaymentMethodPayPay || *got.FeeRateBP != 500 || *got.Fee != 77 || *got.SellerProceeds != 1473 ||
		!got.IdentityDisclosedAt.Equal(t0) || !got.SellerAcknowledgedAt.Equal(t1) || !got.CompletedAt.Equal(t1) ||
		!got.BuyerCancelAgreedAt.Equal(t1) || got.SellerCancelAgreedAt != nil || *got.HandoverMethod != domain.HandoverMethodQR {
		t.Errorf("完了した取引: got = %+v", got)
	}

	if _, err := txs.FindByID(ctx, "no-such-id"); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("存在しない取引: err = %v, want ErrNotFound", err)
	}
}

// 二重処理の最終防衛線として、部分一意インデックスが効いていることを確かめる（DEAL-1, DB）。
func TestTransactionRepository_一意(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	f := newDealFixture(t, gdb)
	txs := repository.NewTransactionRepository(gdb)

	first := f.newRequest(f.buyer)
	if err := txs.Create(ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := txs.Create(ctx, f.newRequest(f.buyer)); !errors.Is(err, repository.ErrDuplicate) {
		t.Errorf("同じ購入者の購入希望が2件: err = %v, want ErrDuplicate", err)
	}
	if err := txs.Create(ctx, f.newRequest(f.buyer2)); err != nil {
		t.Errorf("ほかの購入者の購入希望: err = %v", err)
	}

	// 前の購入希望が期限切れになれば、同じ購入者がもう一度出せる
	if err := gdb.Exec("UPDATE transactions SET status = 'expired' WHERE id = ?", first.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := txs.Create(ctx, f.newRequest(f.buyer)); err != nil {
		t.Errorf("期限切れの後の購入希望: err = %v", err)
	}

	// 1つの商品で成立する取引は1件だけ
	for i, status := range []domain.TransactionStatus{domain.TransactionStatusInDeal, domain.TransactionStatusCompleted} {
		tr := f.newRequest(f.buyer2)
		tr.Status = status
		tr.FeeRateBP, tr.Fee, tr.SellerProceeds = ptr(500), ptr[int64](77), ptr[int64](1473)
		err := txs.Create(ctx, tr)
		if i == 0 && err != nil {
			t.Fatalf("1件目の取引中: err = %v", err)
		}
		if i == 1 && !errors.Is(err, repository.ErrDuplicate) {
			t.Errorf("取引中の商品に完了した取引: err = %v, want ErrDuplicate", err)
		}
	}
}

// transactions・payments の CHECK と外部キーが効いていることを確かめる（DATA, PAY-2）。
func TestTransactionPaymentの制約(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	f := newDealFixture(t, gdb)
	txs := repository.NewTransactionRepository(gdb)
	payments := repository.NewPaymentRepository(gdb)

	base := f.newRequest(f.buyer)
	if err := txs.Create(ctx, base); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		run  func() error
	}{
		{"購入者と出品者が同じ", func() error {
			tr := f.newRequest(f.seller)
			return txs.Create(ctx, tr)
		}},
		{"手数料率だけが入っている", func() error {
			tr := f.newRequest(f.buyer2)
			tr.FeeRateBP = ptr(500)
			return txs.Create(ctx, tr)
		}},
		{"手数料率が100%を超える", func() error {
			tr := f.newRequest(f.buyer2)
			tr.FeeRateBP, tr.Fee, tr.SellerProceeds = ptr(10001), ptr[int64](0), ptr[int64](0)
			return txs.Create(ctx, tr)
		}},
		{"取引のステータスが用語集にない値", func() error {
			tr := f.newRequest(f.buyer2)
			tr.Status = "unknown"
			return txs.Create(ctx, tr)
		}},
		{"取引の支払い方法が用語集にない値", func() error {
			tr := f.newRequest(f.buyer2)
			tr.PaymentMethod = "cash"
			return txs.Create(ctx, tr)
		}},
		{"存在しない商品の取引（外部キー）", func() error {
			tr := f.newRequest(f.buyer2)
			tr.ItemID = "no-such-item"
			return txs.Create(ctx, tr)
		}},
		{"決済の種別が用語集にない値", func() error {
			return payments.Create(ctx, &domain.Payment{TransactionID: base.ID, Kind: "unknown", Amount: 1550,
				Status: domain.PaymentStatusSucceeded, CreatedAt: t0, UpdatedAt: t0})
		}},
		{"決済のステータスが用語集にない値", func() error {
			return payments.Create(ctx, &domain.Payment{TransactionID: base.ID, Kind: domain.PaymentKindAuthorize, Amount: 1550,
				Status: "unknown", CreatedAt: t0, UpdatedAt: t0})
		}},
		{"決済の金額が負", func() error {
			return payments.Create(ctx, &domain.Payment{TransactionID: base.ID, Kind: domain.PaymentKindAuthorize, Amount: -1,
				Status: domain.PaymentStatusSucceeded, CreatedAt: t0, UpdatedAt: t0})
		}},
		{"存在しない取引の決済（外部キー）", func() error {
			return payments.Create(ctx, &domain.Payment{TransactionID: "no-such-tx", Kind: domain.PaymentKindAuthorize, Amount: 1550,
				Status: domain.PaymentStatusSucceeded, CreatedAt: t0, UpdatedAt: t0})
		}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); err == nil {
				t.Error("エラーにならない")
			}
		})
	}
}

func TestPaymentRepository(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	f := newDealFixture(t, gdb)
	tr := f.newRequest(f.buyer)
	if err := repository.NewTransactionRepository(gdb).Create(ctx, tr); err != nil {
		t.Fatal(err)
	}
	payments := repository.NewPaymentRepository(gdb)

	auth := &domain.Payment{
		TransactionID: tr.ID, Kind: domain.PaymentKindAuthorize, PaymentMethod: ptr(domain.PaymentMethodCard),
		Amount: 1550, Status: domain.PaymentStatusSucceeded, CreatedAt: t0, UpdatedAt: t0,
	}
	t1 := t0.Add(time.Hour)
	settle := &domain.Payment{
		TransactionID: tr.ID, Kind: domain.PaymentKindSettle, Amount: 1473,
		Status: domain.PaymentStatusPending, ExternalID: ptr("ext_1"), CreatedAt: t1, UpdatedAt: t1,
	}
	// 新しい方から保存しても、古い順に返る
	for _, p := range []*domain.Payment{settle, auth} {
		if err := payments.Create(ctx, p); err != nil {
			t.Fatal(err)
		}
		if p.ID == "" {
			t.Fatal("ID が採番されていない")
		}
	}

	got, err := payments.ListByTransaction(ctx, tr.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if g := got[0]; g.ID != auth.ID || g.Kind != domain.PaymentKindAuthorize || *g.PaymentMethod != domain.PaymentMethodCard ||
		g.Amount != 1550 || g.Status != domain.PaymentStatusSucceeded || g.ExternalID != nil || !g.CreatedAt.Equal(t0) {
		t.Errorf("枠の確保: got = %+v", g)
	}
	if g := got[1]; g.ID != settle.ID || g.Kind != domain.PaymentKindSettle || g.PaymentMethod != nil ||
		g.Amount != 1473 || g.Status != domain.PaymentStatusPending || *g.ExternalID != "ext_1" {
		t.Errorf("売上の確定: got = %+v", g)
	}

	if got, err := payments.ListByTransaction(ctx, "no-such-tx"); err != nil || len(got) != 0 {
		t.Errorf("記録のない取引: got = %v, err = %v", got, err)
	}
}
