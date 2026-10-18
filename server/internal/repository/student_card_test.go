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

func newStudentCard(userID, objectKey string, at time.Time) *domain.StudentCard {
	return &domain.StudentCard{
		UserID: userID, ObjectKey: objectKey, ContentType: "image/jpeg", SizeBytes: 1024,
		Status: domain.StudentCardStatusPending, SubmittedAt: at, CreatedAt: at, UpdatedAt: at,
	}
}

func TestStudentCardRepository(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	users := repository.NewUserRepository(gdb)
	cards := repository.NewStudentCardRepository(gdb)

	u := newUser("", email, "26238001")
	if err := users.Create(ctx, u); err != nil {
		t.Fatal(err)
	}

	if _, err := cards.FindCurrentByUser(ctx, u.ID); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("登録前: err = %v, want ErrNotFound", err)
	}
	if ok, err := cards.SupersedeCurrent(ctx, u.ID, t0); err != nil || ok {
		t.Fatalf("現在の学生証がないのに置き換えた: ok=%v err=%v", ok, err)
	}

	first := newStudentCard(u.ID, u.ID+"/a", t0)
	if err := cards.Create(ctx, first); err != nil {
		t.Fatal(err)
	}
	got, err := cards.FindCurrentByUser(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != first.ID || got.ObjectKey != u.ID+"/a" || got.Status != domain.StudentCardStatusPending ||
		!got.SubmittedAt.Equal(t0) || got.SizeBytes != 1024 || got.SupersededAt != nil || got.ReviewedAt != nil || got.DeleteAfter != nil {
		t.Errorf("got = %+v", got)
	}
	if ok, err := cards.ExistsByObjectKey(ctx, u.ID+"/a"); err != nil || !ok {
		t.Errorf("登録済みのオブジェクト名: ok=%v err=%v", ok, err)
	}

	// 現在の学生証は1人1件（部分一意インデックス）
	if err := cards.Create(ctx, newStudentCard(u.ID, u.ID+"/b", t0)); !errors.Is(err, repository.ErrDuplicate) {
		t.Errorf("現在の学生証が2件目: err = %v, want ErrDuplicate", err)
	}

	// 再提出：前の学生証は置き換えた日時を入れて残る
	t1 := t0.Add(time.Hour)
	if ok, err := cards.SupersedeCurrent(ctx, u.ID, t1); err != nil || !ok {
		t.Fatalf("置き換え: ok=%v err=%v", ok, err)
	}
	second := newStudentCard(u.ID, u.ID+"/b", t1)
	if err := cards.Create(ctx, second); err != nil {
		t.Fatal(err)
	}
	if got, err := cards.FindCurrentByUser(ctx, u.ID); err != nil || got.ID != second.ID {
		t.Errorf("再提出後の現在の学生証: got=%+v err=%v", got, err)
	}
	if ok, err := cards.ExistsByObjectKey(ctx, u.ID+"/a"); err != nil || !ok {
		t.Errorf("置き換えた学生証が残っていない: ok=%v err=%v", ok, err)
	}
	if err := cards.Create(ctx, newStudentCard(u.ID, u.ID+"/a", t1)); !errors.Is(err, repository.ErrDuplicate) {
		t.Errorf("同じオブジェクト名: err = %v, want ErrDuplicate", err)
	}

	// 承認済みは置き換えない
	if err := gdb.Exec("UPDATE student_cards SET status = 'approved' WHERE id = ?", second.ID).Error; err != nil {
		t.Fatal(err)
	}
	if ok, err := cards.SupersedeCurrent(ctx, u.ID, t1); err != nil || ok {
		t.Errorf("承認済みを置き換えた: ok=%v err=%v", ok, err)
	}
}

func TestUserRepositoryUpdateSellerStatus(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	users := repository.NewUserRepository(gdb)
	u := newUser("", email, "26238001")
	if err := users.Create(ctx, u); err != nil {
		t.Fatal(err)
	}
	from := []domain.SellerStatus{domain.SellerStatusNone, domain.SellerStatusRejected}
	t1 := t0.Add(time.Minute)
	if err := users.UpdateSellerStatus(ctx, u.ID, from, domain.SellerStatusPending, t1); err != nil {
		t.Fatal(err)
	}
	got, err := users.FindByID(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.SellerStatus != domain.SellerStatusPending || !got.UpdatedAt.Equal(t1) {
		t.Errorf("got = %+v", got)
	}
	// 想定外の状態（pending は from にない）なら ErrConflict
	if err := users.UpdateSellerStatus(ctx, u.ID, from, domain.SellerStatusPending, t1); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("想定外の状態: err = %v, want ErrConflict", err)
	}
	if err := users.UpdateSellerStatus(ctx, "missing", from, domain.SellerStatusPending, t1); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("いないユーザー: err = %v, want ErrConflict", err)
	}
}
