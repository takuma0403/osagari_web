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

func date(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

// 有効期限を過ぎた有効・停止のユーザーだけを失効させる（AUTH-3, ADR 0033）。2回目は ErrConflict。
func TestUserRepository_Expire(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	users := repository.NewUserRepository(gdb)

	active := newUser("", email, "26238001")
	suspended := newUser("", "26238002@edu.cc.saga-u.ac.jp", "26238002")
	suspended.Status = domain.UserStatusSuspended
	deleted := newUser("", "26238003@edu.cc.saga-u.ac.jp", "26238003")
	deleted.Status = domain.UserStatusDeleted
	later := newUser("", "26238004@edu.cc.saga-u.ac.jp", "26238004")
	later.ExpiresOn = date(2030, 4, 1)
	for _, u := range []*domain.User{active, suspended, deleted, later} {
		if err := users.Create(ctx, u); err != nil {
			t.Fatal(err)
		}
	}

	// 有効期限の当日はまだ使える
	if got, err := users.ListToExpire(ctx, date(2030, 3, 31)); err != nil || len(got) != 0 {
		t.Fatalf("期限当日: got=%d err=%v", len(got), err)
	}
	if err := users.Expire(ctx, active.ID, date(2030, 3, 31), t0); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("期限当日の失効: err = %v, want ErrConflict", err)
	}

	today := date(2030, 4, 1)
	got, err := users.ListToExpire(ctx, today)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID == got[1].ID {
		t.Fatalf("失効させる人 = %+v, want 有効と停止の2人", got)
	}
	for _, u := range got {
		if u.ID != active.ID && u.ID != suspended.ID {
			t.Errorf("対象外のユーザー: %+v", u)
		}
	}

	now := t0.Add(time.Hour)
	for _, u := range []*domain.User{active, suspended} {
		if err := users.Expire(ctx, u.ID, today, now); err != nil {
			t.Fatalf("失効: %v", err)
		}
		if got, _ := users.FindByID(ctx, u.ID); got.Status != domain.UserStatusExpired || !got.UpdatedAt.Equal(now) {
			t.Errorf("失効後: got = %+v", got)
		}
		if err := users.Expire(ctx, u.ID, today, now); !errors.Is(err, repository.ErrConflict) {
			t.Errorf("二重の失効: err = %v, want ErrConflict", err)
		}
	}
	if err := users.Expire(ctx, deleted.ID, today, now); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("削除済みの失効: err = %v, want ErrConflict", err)
	}
	if got, _ := users.ListToExpire(ctx, today); len(got) != 0 {
		t.Errorf("失効後の対象 = %d 人, want 0", len(got))
	}
}

// 削除予定日はすべての行（置き換えた行を含む）にまだ入っていなければ入れ、画像を消した日時は一度だけ記録する（ADR 0014, ADR 0033）。
func TestStudentCardRepository_削除予定日と画像の削除(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	users := repository.NewUserRepository(gdb)
	cards := repository.NewStudentCardRepository(gdb)
	u := newUser("", email, "26238001")
	other := newUser("", "26238002@edu.cc.saga-u.ac.jp", "26238002")
	for _, x := range []*domain.User{u, other} {
		if err := users.Create(ctx, x); err != nil {
			t.Fatal(err)
		}
	}
	old := newStudentCard(u.ID, u.ID+"/a", t0)
	if err := cards.Create(ctx, old); err != nil {
		t.Fatal(err)
	}
	if _, err := cards.SupersedeCurrent(ctx, u.ID, t0); err != nil {
		t.Fatal(err)
	}
	cur := newStudentCard(u.ID, u.ID+"/b", t0)
	otherCard := newStudentCard(other.ID, other.ID+"/a", t0)
	for _, c := range []*domain.StudentCard{cur, otherCard} {
		if err := cards.Create(ctx, c); err != nil {
			t.Fatal(err)
		}
	}

	deleteAfter := date(2033, 4, 1)
	if n, err := cards.SetDeleteAfter(ctx, u.ID, deleteAfter, t0); err != nil || n != 2 {
		t.Fatalf("削除予定日: n=%d err=%v, want 2", n, err)
	}
	// すでに入っていれば変えない
	if n, err := cards.SetDeleteAfter(ctx, u.ID, date(2040, 1, 1), t0); err != nil || n != 0 {
		t.Fatalf("2回目の削除予定日: n=%d err=%v, want 0", n, err)
	}
	for _, id := range []string{old.ID, cur.ID} {
		if got, _ := cards.FindByID(ctx, id); got.DeleteAfter == nil || !got.DeleteAfter.Equal(deleteAfter) {
			t.Errorf("削除予定日 = %v, want %v", got.DeleteAfter, deleteAfter)
		}
	}
	if got, _ := cards.FindByID(ctx, otherCard.ID); got.DeleteAfter != nil {
		t.Errorf("ほかの人の学生証に削除予定日が入った: %v", got.DeleteAfter)
	}

	if got, err := cards.ListImagesToDelete(ctx, date(2033, 3, 31)); err != nil || len(got) != 0 {
		t.Fatalf("削除予定日の前日: got=%d err=%v", len(got), err)
	}
	got, err := cards.ListImagesToDelete(ctx, deleteAfter)
	if err != nil || len(got) != 2 {
		t.Fatalf("削除予定日の当日: got=%d err=%v, want 2", len(got), err)
	}

	now := t0.Add(time.Hour)
	if err := cards.MarkImageDeleted(ctx, old.ID, now); err != nil {
		t.Fatal(err)
	}
	if err := cards.MarkImageDeleted(ctx, old.ID, now); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("二重の記録: err = %v, want ErrConflict", err)
	}
	if c, _ := cards.FindByID(ctx, old.ID); c.ImageDeletedAt == nil || !c.ImageDeletedAt.Equal(now) {
		t.Errorf("画像を消した日時 = %v, want %v", c.ImageDeletedAt, now)
	}
	if got, _ := cards.ListImagesToDelete(ctx, deleteAfter); len(got) != 1 || got[0].ID != cur.ID {
		t.Errorf("記録後の対象 = %+v, want 現在の学生証だけ", got)
	}
}

// 当事者の取引をステータスで探し、取引中の取引は同意なしでキャンセルできる（DEAL-4 の例外, ADR 0033）。
func TestTransactionRepository_失効によるキャンセル(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	f := newDealFixture(t, gdb)
	txs := repository.NewTransactionRepository(gdb)
	req := f.newRequest(f.buyer)
	req2 := f.newRequest(f.buyer2)
	for _, tr := range []*domain.Transaction{req, req2} {
		if err := txs.Create(ctx, tr); err != nil {
			t.Fatal(err)
		}
	}

	for _, tt := range []struct {
		name   string
		userID string
		want   int
	}{{"出品者", f.seller.ID, 2}, {"購入者", f.buyer.ID, 1}, {"購入者2", f.buyer2.ID, 1}} {
		got, err := txs.ListByPartyAndStatus(ctx, tt.userID, domain.TransactionStatusRequested)
		if err != nil || len(got) != tt.want {
			t.Errorf("%s の承認待ち: got=%d err=%v, want %d", tt.name, len(got), err, tt.want)
		}
	}

	if err := txs.CancelBySystem(ctx, req.ID, t0); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("承認待ちのキャンセル: err = %v, want ErrConflict", err)
	}
	if err := txs.Approve(ctx, req.ID, f.seller.ID, 500, 77, 1473, t0); err != nil {
		t.Fatal(err)
	}
	if got, _ := txs.ListByPartyAndStatus(ctx, f.buyer.ID, domain.TransactionStatusInDeal); len(got) != 1 || got[0].ID != req.ID {
		t.Errorf("購入者の取引中 = %+v", got)
	}
	t1 := t0.Add(time.Hour)
	if err := txs.CancelBySystem(ctx, req.ID, t1); err != nil {
		t.Fatalf("キャンセル: %v", err)
	}
	got, err := txs.FindByID(ctx, req.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.TransactionStatusCancelled || got.BuyerCancelAgreedAt != nil || got.SellerCancelAgreedAt != nil || !got.UpdatedAt.Equal(t1) {
		t.Errorf("キャンセル後: got = %+v（同意日時は入れない）", got)
	}
	if err := txs.CancelBySystem(ctx, req.ID, t1); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("二重のキャンセル: err = %v, want ErrConflict", err)
	}
}
