package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/repository"
	"github.com/takuma0403/osagari_web/server/internal/testutil/testdb"
)

// 承認の条件付きUPDATE：成功、出品者違い・期限切れ・状態違いは ErrConflict、2件目の成立は ErrDuplicate（DEAL-1, DB）。
func TestTransactionRepository_Approve(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	f := newDealFixture(t, gdb)
	txs := repository.NewTransactionRepository(gdb)

	req, req2 := f.newRequest(f.buyer), f.newRequest(f.buyer2)
	for _, tr := range []*domain.Transaction{req, req2} {
		if err := txs.Create(ctx, tr); err != nil {
			t.Fatal(err)
		}
	}
	deadline := req.ApprovalDeadline

	if err := txs.Approve(ctx, req.ID, f.buyer.ID, 500, 77, 1473, t0); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("出品者でない: err = %v, want ErrConflict", err)
	}
	if err := txs.Approve(ctx, req.ID, f.seller.ID, 500, 77, 1473, deadline); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("承認期限ちょうど: err = %v, want ErrConflict", err)
	}

	now := deadline.Add(-time.Second)
	if err := txs.Approve(ctx, req.ID, f.seller.ID, 500, 77, 1473, now); err != nil {
		t.Fatalf("承認: %v", err)
	}
	got, err := txs.FindByID(ctx, req.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.TransactionStatusInDeal || *got.FeeRateBP != 500 || *got.Fee != 77 || *got.SellerProceeds != 1473 ||
		!got.IdentityDisclosedAt.Equal(now) || !got.UpdatedAt.Equal(now) {
		t.Errorf("承認後: got = %+v", got)
	}

	if err := txs.Approve(ctx, req.ID, f.seller.ID, 500, 77, 1473, now); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("二重承認: err = %v, want ErrConflict", err)
	}
	if err := txs.Approve(ctx, req2.ID, f.seller.ID, 500, 77, 1473, now); !errors.Is(err, repository.ErrDuplicate) {
		t.Errorf("同じ商品の2件目の承認: err = %v, want ErrDuplicate", err)
	}
}

func TestTransactionRepository_UpdateStatusとAcknowledge(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	f := newDealFixture(t, gdb)
	txs := repository.NewTransactionRepository(gdb)
	req := f.newRequest(f.buyer)
	if err := txs.Create(ctx, req); err != nil {
		t.Fatal(err)
	}
	t1 := t0.Add(time.Hour)

	if err := txs.Acknowledge(ctx, req.ID, f.seller.ID, t1); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("承認前の確認: err = %v, want ErrConflict", err)
	}
	if err := txs.UpdateStatus(ctx, req.ID, domain.TransactionStatusInDeal, domain.TransactionStatusCancelled, t1); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("状態違い: err = %v, want ErrConflict", err)
	}
	if err := txs.UpdateStatus(ctx, req.ID, domain.TransactionStatusRequested, domain.TransactionStatusRejected, t1); err != nil {
		t.Fatalf("却下: %v", err)
	}
	if got, _ := txs.FindByID(ctx, req.ID); got.Status != domain.TransactionStatusRejected || !got.UpdatedAt.Equal(t1) {
		t.Errorf("却下後: got = %+v", got)
	}
	if err := txs.UpdateStatus(ctx, req.ID, domain.TransactionStatusRequested, domain.TransactionStatusRejected, t1); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("二重の却下: err = %v, want ErrConflict", err)
	}

	deal := f.newRequest(f.buyer2)
	if err := txs.Create(ctx, deal); err != nil {
		t.Fatal(err)
	}
	if err := txs.Approve(ctx, deal.ID, f.seller.ID, 500, 77, 1473, t0); err != nil {
		t.Fatal(err)
	}
	if err := txs.Acknowledge(ctx, deal.ID, f.buyer2.ID, t1); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("購入者の確認: err = %v, want ErrConflict", err)
	}
	if err := txs.Acknowledge(ctx, deal.ID, f.seller.ID, t1); err != nil {
		t.Fatalf("確認: %v", err)
	}
	if got, _ := txs.FindByID(ctx, deal.ID); !got.SellerAcknowledgedAt.Equal(t1) {
		t.Errorf("確認後: got = %+v", got)
	}
	if err := txs.Acknowledge(ctx, deal.ID, f.seller.ID, t1.Add(time.Hour)); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("二重の確認: err = %v, want ErrConflict", err)
	}
}

func TestTransactionRepository_一覧と件数(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	f := newDealFixture(t, gdb)
	txs := repository.NewTransactionRepository(gdb)

	// 新しい方から作っても、希望の古い順に返る
	late, early := f.newRequest(f.buyer), f.newRequest(f.buyer2)
	late.CreatedAt = t0.Add(time.Hour)
	for _, tr := range []*domain.Transaction{late, early} {
		if err := txs.Create(ctx, tr); err != nil {
			t.Fatal(err)
		}
	}
	// 却下した購入希望は含めない
	rejected := f.newRequest(f.seller)
	rejected.BuyerID, rejected.Status = f.buyer.ID, domain.TransactionStatusRejected
	if err := txs.Create(ctx, rejected); err != nil {
		t.Fatal(err)
	}

	got, err := txs.ListRequestedByItem(ctx, f.item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != early.ID || got[1].ID != late.ID {
		t.Errorf("購入希望の一覧: got = %+v", got)
	}
	if n, err := txs.CountRequestedByItem(ctx, f.item.ID); err != nil || n != 2 {
		t.Errorf("購入希望の件数 = %d, %v, want 2", n, err)
	}

	// 購入者ごとの承認待ちの購入希望。却下した購入希望だけの人は見つからない
	if mine, err := txs.FindRequestedByItemAndBuyer(ctx, f.item.ID, f.buyer.ID); err != nil || mine.ID != late.ID {
		t.Errorf("購入者の購入希望: got = %+v, %v, want %s", mine, err, late.ID)
	}
	if _, err := txs.FindRequestedByItemAndBuyer(ctx, f.item.ID, f.seller.ID); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("購入希望を出していない人: err = %v, want ErrNotFound", err)
	}

	// 取引完了の回数は購入者・出品者の両方で数える
	done := f.newRequest(f.buyer)
	done.Status = domain.TransactionStatusCompleted
	done.FeeRateBP, done.Fee, done.SellerProceeds = ptr(500), ptr[int64](77), ptr[int64](1473)
	if err := txs.Create(ctx, done); err != nil {
		t.Fatal(err)
	}
	counts, err := txs.CountCompletedByUsers(ctx, []string{f.seller.ID, f.buyer.ID, f.buyer2.ID})
	if err != nil {
		t.Fatal(err)
	}
	if counts[f.seller.ID] != 1 || counts[f.buyer.ID] != 1 || counts[f.buyer2.ID] != 0 {
		t.Errorf("取引完了の回数: got = %v", counts)
	}
}

// 取引の進み具合による商品のステータスの変更は、公開日時を変えない（ADR 0022）。
func TestItemRepository_UpdateDealStatus(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	f := newDealFixture(t, gdb)
	items := repository.NewItemRepository(gdb)
	t1 := t0.Add(time.Hour)

	if err := items.UpdateDealStatus(ctx, f.item.ID, f.seller.ID, []domain.ItemStatus{domain.ItemStatusRequested}, domain.ItemStatusInDeal, t1); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("状態違い: err = %v, want ErrConflict", err)
	}
	if err := items.UpdateDealStatus(ctx, f.item.ID, f.buyer.ID, []domain.ItemStatus{domain.ItemStatusPublished}, domain.ItemStatusRequested, t1); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("出品者違い: err = %v, want ErrConflict", err)
	}
	for _, to := range []domain.ItemStatus{domain.ItemStatusRequested, domain.ItemStatusPublished} {
		from := []domain.ItemStatus{domain.ItemStatusPublished, domain.ItemStatusRequested}
		if err := items.UpdateDealStatus(ctx, f.item.ID, f.seller.ID, from, to, t1); err != nil {
			t.Fatalf("%s: %v", to, err)
		}
		got, err := items.FindByID(ctx, f.item.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != to || !got.PublishedAt.Equal(t0) || !got.UpdatedAt.Equal(t1) {
			t.Errorf("%s: status = %s, publishedAt = %v, updatedAt = %v", to, got.Status, got.PublishedAt, got.UpdatedAt)
		}
	}
}
