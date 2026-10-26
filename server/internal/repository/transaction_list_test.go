package repository_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/repository"
	"github.com/takuma0403/osagari_web/server/internal/testutil/testdb"
)

// 取引一覧の読み出し：当事者・ステータスでの絞り込み、最後に動いた順、続きの位置、商品ごとの購入希望の件数（ADR 0028）。
func TestTransactionRepository_取引一覧(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	f := newDealFixture(t, gdb)
	items := repository.NewItemRepository(gdb)
	txs := repository.NewTransactionRepository(gdb)

	item2 := &domain.Item{
		SellerID: f.seller.ID, ShareID: "share0000002", Status: domain.ItemStatusPublished,
		ItemContent: domain.ItemContent{Title: "微分積分", Price: ptr[int64](1200)},
		Images:      []domain.ItemImage{newItemImage(f.seller.ID + "/b"), newItemImage(f.seller.ID + "/a")},
		PublishedAt: &t0, CreatedAt: t0, UpdatedAt: t0,
	}
	if err := items.Create(ctx, item2); err != nil {
		t.Fatal(err)
	}

	// create は t0 から h 時間後に作った（最後に動いた）status の取引を保存して返す
	create := func(itemID string, buyer *domain.User, status domain.TransactionStatus, h time.Duration) *domain.Transaction {
		t.Helper()
		tr := f.newRequest(buyer)
		tr.ItemID, tr.Status = itemID, status
		tr.CreatedAt = t0.Add(h * time.Hour)
		tr.UpdatedAt = tr.CreatedAt
		if err := txs.Create(ctx, tr); err != nil {
			t.Fatal(err)
		}
		return tr
	}
	reqA := create(f.item.ID, f.buyer, domain.TransactionStatusRequested, 0)
	reqB := create(f.item.ID, f.buyer2, domain.TransactionStatusRequested, 1)
	reqC := create(item2.ID, f.buyer, domain.TransactionStatusRequested, 2)
	done := create(item2.ID, f.buyer2, domain.TransactionStatusCompleted, 3)
	cancelled := create(f.item.ID, f.buyer, domain.TransactionStatusCancelled, 4)
	expired := create(f.item.ID, f.buyer, domain.TransactionStatusExpired, 4)

	ids := func(trs []*domain.Transaction) []string {
		out := make([]string, len(trs))
		for i, tr := range trs {
			out[i] = tr.ID
		}
		return out
	}
	// 同じ日時なら ID の大きい順
	sameTime := []string{cancelled.ID, expired.ID}
	slices.Sort(sameTime)
	slices.Reverse(sameTime)

	ended := []domain.TransactionStatus{domain.TransactionStatusRejected, domain.TransactionStatusExpired, domain.TransactionStatusCancelled, domain.TransactionStatusCompleted}
	tests := []struct {
		name     string
		role     domain.TransactionRole
		userID   string
		statuses []domain.TransactionStatus
		after    *domain.TransactionCursor
		limit    int
		want     []string
	}{
		{"購入者の進行中（本人の分だけ、新しい順）", domain.TransactionRoleBuyer, f.buyer.ID,
			[]domain.TransactionStatus{domain.TransactionStatusRequested, domain.TransactionStatusInDeal}, nil, 0, []string{reqC.ID, reqA.ID}},
		{"購入者の終了（件数の上限）", domain.TransactionRoleBuyer, f.buyer.ID, ended, nil, 1, sameTime[:1]},
		{"購入者の終了（続きの位置の後ろ）", domain.TransactionRoleBuyer, f.buyer.ID, ended,
			&domain.TransactionCursor{UpdatedAt: t0.Add(4 * time.Hour), ID: sameTime[0]}, 0, sameTime[1:]},
		{"出品者の終了（キャンセル・完了だけ）", domain.TransactionRoleSeller, f.seller.ID,
			[]domain.TransactionStatus{domain.TransactionStatusCancelled, domain.TransactionStatusCompleted}, nil, 0, []string{cancelled.ID, done.ID}},
		{"出品者として当事者でない", domain.TransactionRoleSeller, f.buyer.ID, ended, nil, 0, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := txs.ListByParty(ctx, tt.role, tt.userID, tt.statuses, tt.after, tt.limit)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(ids(got), tt.want) {
				t.Errorf("got = %v, want %v", ids(got), tt.want)
			}
		})
	}

	groups, err := txs.ListRequestGroupsBySeller(ctx, f.seller.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.TransactionRequestGroup{
		{ItemID: item2.ID, Count: 1, LatestRequestedAt: reqC.CreatedAt},
		{ItemID: f.item.ID, Count: 2, LatestRequestedAt: reqB.CreatedAt},
	}
	if len(groups) != len(want) {
		t.Fatalf("商品ごとの購入希望: got = %+v, want %+v", groups, want)
	}
	for i := range want {
		if groups[i].ItemID != want[i].ItemID || groups[i].Count != want[i].Count || !groups[i].LatestRequestedAt.Equal(want[i].LatestRequestedAt) {
			t.Errorf("商品ごとの購入希望[%d]: got = %+v, want %+v", i, groups[i], want[i])
		}
	}
	if groups, err := txs.ListRequestGroupsBySeller(ctx, f.buyer.ID); err != nil || len(groups) != 0 {
		t.Errorf("出品していない人: groups = %+v, err = %v", groups, err)
	}
}

// 商品をまとめて読む：1枚目の写真だけを付け、見つからない ID は含めない（ADR 0028）。
func TestItemRepository_FindByIDs(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	f := newDealFixture(t, gdb)
	items := repository.NewItemRepository(gdb)
	withImages := &domain.Item{
		SellerID: f.seller.ID, ShareID: "share0000002", Status: domain.ItemStatusInDeal,
		ItemContent: domain.ItemContent{Title: "微分積分", Price: ptr[int64](1200)},
		Images:      []domain.ItemImage{newItemImage(f.seller.ID + "/b"), newItemImage(f.seller.ID + "/a")},
		PublishedAt: &t0, CreatedAt: t0, UpdatedAt: t0,
	}
	if err := items.Create(ctx, withImages); err != nil {
		t.Fatal(err)
	}

	got, err := items.FindByIDs(ctx, []string{f.item.ID, withImages.ID, "missing"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got = %v, want 2件", got)
	}
	if it := got[f.item.ID]; it.Title != "線形代数" || len(it.Images) != 0 {
		t.Errorf("写真なし: got = %+v", it)
	}
	if it := got[withImages.ID]; it.Status != domain.ItemStatusInDeal || len(it.Images) != 1 || it.Images[0].ObjectKey != f.seller.ID+"/b" {
		t.Errorf("写真あり: got = %+v", it)
	}
	if got, err := items.FindByIDs(ctx, nil); err != nil || len(got) != 0 {
		t.Errorf("空: got = %v, err = %v", got, err)
	}
}
