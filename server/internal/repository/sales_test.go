package repository_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/repository"
	"github.com/takuma0403/osagari_web/server/internal/testutil/testdb"
)

// 出品者の商品の一覧：ステータスを問わず本人の商品だけを、最後に更新した順に1枚目の写真つきで返す（AUTH-3, ADR 0029）。
func TestItemRepository_ListBySeller(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	f := newDealFixture(t, gdb)
	items := repository.NewItemRepository(gdb)

	create := func(seller *domain.User, shareID string, status domain.ItemStatus, h time.Duration, imageKeys ...string) *domain.Item {
		t.Helper()
		it := &domain.Item{
			SellerID: seller.ID, ShareID: shareID, Status: status,
			ItemContent: domain.ItemContent{Title: shareID}, CreatedAt: t0, UpdatedAt: t0.Add(h * time.Hour),
		}
		for _, k := range imageKeys {
			it.Images = append(it.Images, newItemImage(seller.ID+"/"+k))
		}
		if err := items.Create(ctx, it); err != nil {
			t.Fatal(err)
		}
		return it
	}
	draft := create(f.seller, "share0000002", domain.ItemStatusDraft, 2)
	withdrawn := create(f.seller, "share0000003", domain.ItemStatusWithdrawn, 3, "b", "a")
	create(f.buyer, "share0000004", domain.ItemStatusPublished, 4) // ほかの人の出品は出ない

	got, err := items.ListBySeller(ctx, f.seller.ID)
	if err != nil {
		t.Fatal(err)
	}
	gotIDs := make([]string, len(got))
	for i, it := range got {
		gotIDs[i] = it.ID
	}
	if want := []string{withdrawn.ID, draft.ID, f.item.ID}; !slices.Equal(gotIDs, want) {
		t.Fatalf("並び = %v, want %v", gotIDs, want)
	}
	if imgs := got[0].Images; len(imgs) != 1 || imgs[0].ObjectKey != f.seller.ID+"/b" {
		t.Errorf("サムネイル = %+v, want 1枚目だけ", imgs)
	}
	if len(got[1].Images) != 0 {
		t.Errorf("写真のない商品の Images = %+v", got[1].Images)
	}

	none, err := items.ListBySeller(ctx, "missing")
	if err != nil || len(none) != 0 {
		t.Errorf("出品のないユーザー: got = %v, err = %v", none, err)
	}
}

// 出金の記録は取引ごとに1行だけで、取引IDごとに引ける（ADR 0029）。
func TestPaymentRepository_出金(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	f := newDealFixture(t, gdb)
	txs := repository.NewTransactionRepository(gdb)
	payments := repository.NewPaymentRepository(gdb)

	var trs []*domain.Transaction
	for _, b := range []*domain.User{f.buyer, f.buyer2} {
		tr := f.newRequest(b)
		tr.Status = domain.TransactionStatusRejected
		if err := txs.Create(ctx, tr); err != nil {
			t.Fatal(err)
		}
		trs = append(trs, tr)
	}
	record := func(tr *domain.Transaction, kind domain.PaymentKind, at time.Time) error {
		return payments.Create(ctx, &domain.Payment{
			TransactionID: tr.ID, Kind: kind, Amount: 1473, Status: domain.PaymentStatusSucceeded, CreatedAt: at, UpdatedAt: at,
		})
	}
	if err := record(trs[0], domain.PaymentKindSettle, t0); err != nil {
		t.Fatal(err)
	}
	if err := record(trs[0], domain.PaymentKindPayout, t0.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := record(trs[0], domain.PaymentKindPayout, t0.Add(2*time.Hour)); !errors.Is(err, repository.ErrDuplicate) {
		t.Errorf("2回目の出金: err = %v, want ErrDuplicate", err)
	}
	if err := record(trs[1], domain.PaymentKindSettle, t0); err != nil {
		t.Fatal(err)
	}

	got, err := payments.ListPayoutsByTransactions(ctx, []string{trs[0].ID, trs[1].ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[trs[0].ID] == nil || !got[trs[0].ID].CreatedAt.Equal(t0.Add(time.Hour)) {
		t.Errorf("出金の記録 = %+v", got)
	}
	if empty, err := payments.ListPayoutsByTransactions(ctx, nil); err != nil || len(empty) != 0 {
		t.Errorf("空の入力: got = %v, err = %v", empty, err)
	}
}
