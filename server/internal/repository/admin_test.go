package repository_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/repository"
	"github.com/takuma0403/osagari_web/server/internal/testutil/testdb"
)

// 管理画面のユーザーの一覧：学籍番号の前方一致、状態・退学申請での絞り込み、登録の新しい順と続きの位置（ADMIN, ADR 0034）。
func TestUserRepository_管理画面の一覧(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	users := repository.NewUserRepository(gdb)

	// create は t0 から h 時間後に登録した status のユーザーを保存して返す
	create := func(number string, status domain.UserStatus, h time.Duration, withdrawal bool) *domain.User {
		t.Helper()
		u := newUser("", number+"@edu.cc.saga-u.ac.jp", number)
		u.Status = status
		u.CreatedAt = t0.Add(h * time.Hour)
		if withdrawal {
			at := t0.Add(10 * time.Hour)
			u.WithdrawalRequestedAt = &at
		}
		if err := users.Create(ctx, u); err != nil {
			t.Fatal(err)
		}
		return u
	}
	a := create("26238001", domain.UserStatusActive, 0, false)
	b := create("26238002", domain.UserStatusSuspended, 1, true)
	c := create("25141003", domain.UserStatusExpired, 2, false)
	d := create("26238004", domain.UserStatusActive, 3, true)
	create("26238005", domain.UserStatusDeleted, 4, true) // 削除済みは申請中に数えない

	ids := func(us []*domain.User) []string {
		out := make([]string, len(us))
		for i, u := range us {
			out[i] = u.ID
		}
		return out
	}
	list := func(q domain.AdminUserQuery, after *domain.AdminCursor, limit int) []string {
		t.Helper()
		got, err := users.ListForAdmin(ctx, q, after, limit)
		if err != nil {
			t.Fatal(err)
		}
		return ids(got)
	}
	eq := func(name string, got []string, want ...*domain.User) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("%s: got %v, want %v", name, got, ids(want))
		}
		for i := range want {
			if got[i] != want[i].ID {
				t.Fatalf("%s: got %v, want %v", name, got, ids(want))
			}
		}
	}

	all := domain.AdminUserQuery{Filter: domain.AdminUserFilterAll}
	if got := list(all, nil, 10); len(got) != 5 {
		t.Errorf("すべて: %d 件, want 5", len(got))
	}
	eq("前方一致", list(domain.AdminUserQuery{StudentNumberPrefix: "26238", Filter: domain.AdminUserFilterActive}, nil, 10), d, a)
	eq("停止", list(domain.AdminUserQuery{Filter: domain.AdminUserFilterSuspended}, nil, 10), b)
	eq("失効", list(domain.AdminUserQuery{Filter: domain.AdminUserFilterExpired}, nil, 10), c)
	eq("退学申請", list(domain.AdminUserQuery{Filter: domain.AdminUserFilterWithdrawalRequested}, nil, 10), d, b)
	eq("続きの位置", list(all, &domain.AdminCursor{At: d.CreatedAt, ID: d.ID}, 2), c, b)

	if n, err := users.CountWithdrawalRequested(ctx); err != nil || n != 2 {
		t.Errorf("退学申請の件数: n=%d err=%v, want 2", n, err)
	}
	got, err := users.FindByID(ctx, b.ID)
	if err != nil || got.WithdrawalRequestedAt == nil || !got.WithdrawalRequestedAt.Equal(t0.Add(10*time.Hour)) {
		t.Errorf("退学の申請日時を読み戻せない: %+v err=%v", got, err)
	}
}

// 状態と有効期限は、状態が想定どおりのときだけ変える（条件付きUPDATE。DB）。
func TestUserRepository_状態と有効期限の変更(t *testing.T) {
	ctx := context.Background()
	users := repository.NewUserRepository(testdb.New(t))
	u := newUser("", email, "26238001")
	if err := users.Create(ctx, u); err != nil {
		t.Fatal(err)
	}
	now := t0.Add(time.Hour)
	active := []domain.UserStatus{domain.UserStatusActive}

	if err := users.UpdateStatus(ctx, u.ID, active, domain.UserStatusSuspended, now); err != nil {
		t.Fatal(err)
	}
	if err := users.UpdateStatus(ctx, u.ID, active, domain.UserStatusSuspended, now); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("停止中を停止: %v, want ErrConflict", err)
	}
	if err := users.UpdateStatus(ctx, "missing", active, domain.UserStatusSuspended, now); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("いないユーザー: %v, want ErrConflict", err)
	}

	newDate := date(2031, 3, 31)
	if err := users.UpdateExpiresOn(ctx, u.ID, active, newDate, now); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("停止中を有効だけの条件で変更: %v, want ErrConflict", err)
	}
	if err := users.UpdateExpiresOn(ctx, u.ID, domain.AdminClosableUserStatuses, newDate, now); err != nil {
		t.Fatal(err)
	}
	got, err := users.FindByID(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.UserStatusSuspended || !got.ExpiresOn.Equal(newDate) || !got.UpdatedAt.Equal(now) {
		t.Errorf("got=%+v", got)
	}
}

// キャンセル回数は双方の同意でキャンセルになった取引だけを両方の当事者に数え、取引中の件数も当事者ごとに数える（DEAL-4, ADR 0034）。
func TestTransactionRepository_キャンセル回数と取引中の件数(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	f := newDealFixture(t, gdb)
	txs := repository.NewTransactionRepository(gdb)

	items := repository.NewItemRepository(gdb)
	at := t0.Add(time.Hour)
	// 取引中は商品ごとに1件だけなので、取引ごとに商品を作る
	n := 0
	create := func(buyer *domain.User, status domain.TransactionStatus, buyerAgreed, sellerAgreed bool) {
		t.Helper()
		n++
		it := &domain.Item{
			SellerID: f.seller.ID, ShareID: fmt.Sprintf("share%07d", 100+n), Status: domain.ItemStatusPublished,
			ItemContent: domain.ItemContent{Title: "線形代数", Price: ptr[int64](1550)},
			PublishedAt: &t0, CreatedAt: t0, UpdatedAt: t0,
		}
		if err := items.Create(ctx, it); err != nil {
			t.Fatal(err)
		}
		tr := f.newRequest(buyer)
		tr.ItemID, tr.Status = it.ID, status
		if buyerAgreed {
			tr.BuyerCancelAgreedAt = &at
		}
		if sellerAgreed {
			tr.SellerCancelAgreedAt = &at
		}
		if err := txs.Create(ctx, tr); err != nil {
			t.Fatal(err)
		}
	}
	create(f.buyer, domain.TransactionStatusCancelled, true, true)    // 双方の同意：数える
	create(f.buyer2, domain.TransactionStatusCancelled, true, false)  // 管理者のキャンセル（申請中だった）：数えない
	create(f.buyer2, domain.TransactionStatusCancelled, false, false) // 失効・削除のキャンセル：数えない
	create(f.buyer2, domain.TransactionStatusInDeal, true, false)     // 申請中の取引中：数えない
	create(f.buyer, domain.TransactionStatusInDeal, false, false)

	ids := []string{f.seller.ID, f.buyer.ID, f.buyer2.ID}
	cancels, err := txs.CountCancelledByUsers(ctx, ids)
	if err != nil {
		t.Fatal(err)
	}
	if cancels[f.seller.ID] != 1 || cancels[f.buyer.ID] != 1 || cancels[f.buyer2.ID] != 0 {
		t.Errorf("キャンセル回数 = %v", cancels)
	}
	deals, err := txs.CountInDealByUsers(ctx, ids)
	if err != nil {
		t.Fatal(err)
	}
	if deals[f.seller.ID] != 2 || deals[f.buyer.ID] != 1 || deals[f.buyer2.ID] != 1 {
		t.Errorf("取引中の件数 = %v", deals)
	}
}

// 管理画面の取引・出品の一覧は、すべてを新しい順に返し、続きの位置から読める（ADMIN）。
func TestRepository_管理画面の取引と出品の一覧(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	f := newDealFixture(t, gdb)
	txs := repository.NewTransactionRepository(gdb)
	items := repository.NewItemRepository(gdb)

	var trs []*domain.Transaction
	for i, s := range []domain.TransactionStatus{domain.TransactionStatusRejected, domain.TransactionStatusInDeal, domain.TransactionStatusRequested} {
		tr := f.newRequest(f.buyer)
		tr.Status = s
		tr.UpdatedAt = t0.Add(time.Duration(i) * time.Hour)
		if err := txs.Create(ctx, tr); err != nil {
			t.Fatal(err)
		}
		trs = append(trs, tr)
	}
	got, err := txs.ListForAdmin(ctx, nil, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != trs[2].ID || got[1].ID != trs[1].ID {
		t.Fatalf("取引の1ページ目 = %v", got)
	}
	got, err = txs.ListForAdmin(ctx, &domain.AdminCursor{At: got[1].UpdatedAt, ID: got[1].ID}, 2)
	if err != nil || len(got) != 1 || got[0].ID != trs[0].ID {
		t.Fatalf("取引の続き = %v err=%v", got, err)
	}

	draft := &domain.Item{
		SellerID: f.seller.ID, ShareID: "share0000002", Status: domain.ItemStatusDraft,
		ItemContent: domain.ItemContent{Title: "下書き"},
		Images:      []domain.ItemImage{newItemImage(f.seller.ID + "/a")},
		CreatedAt:   t0.Add(time.Hour), UpdatedAt: t0.Add(time.Hour),
	}
	if err := items.Create(ctx, draft); err != nil {
		t.Fatal(err)
	}
	its, err := items.ListForAdmin(ctx, nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(its) != 2 || its[0].ID != draft.ID || its[1].ID != f.item.ID || len(its[0].Images) != 1 {
		t.Fatalf("出品の一覧 = %+v", its)
	}
	its, err = items.ListForAdmin(ctx, &domain.AdminCursor{At: draft.CreatedAt, ID: draft.ID}, 10)
	if err != nil || len(its) != 1 || its[0].ID != f.item.ID {
		t.Fatalf("出品の続き = %+v err=%v", its, err)
	}
}
