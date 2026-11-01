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

// 審査待ちの一覧・条件付きの審査・閲覧記録（ADMIN, CARD）。
func TestStudentCardRepository_Review(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	users := repository.NewUserRepository(gdb)
	cards := repository.NewStudentCardRepository(gdb)
	views := repository.NewStudentCardViewRepository(gdb)

	var us []*domain.User
	for _, n := range []string{"26238001", "26238002", "26238003"} {
		u := newUser("", n+"@edu.cc.saga-u.ac.jp", n)
		if err := users.Create(ctx, u); err != nil {
			t.Fatal(err)
		}
		us = append(us, u)
	}
	admin := us[2]

	// us[0]：古い提出を置き換えて再提出した（新しい方だけが一覧に出る）。us[1]：us[0] の再提出より前に提出した
	old := newStudentCard(us[0].ID, us[0].ID+"/old", t0)
	if err := cards.Create(ctx, old); err != nil {
		t.Fatal(err)
	}
	b := newStudentCard(us[1].ID, us[1].ID+"/b", t0.Add(time.Hour))
	if err := cards.Create(ctx, b); err != nil {
		t.Fatal(err)
	}
	if ok, err := cards.SupersedeCurrent(ctx, us[0].ID, t0.Add(2*time.Hour)); err != nil || !ok {
		t.Fatalf("置き換え: ok=%v err=%v", ok, err)
	}
	a := newStudentCard(us[0].ID, us[0].ID+"/a", t0.Add(2*time.Hour))
	if err := cards.Create(ctx, a); err != nil {
		t.Fatal(err)
	}

	pending, err := cards.ListPending(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 2 || pending[0].ID != b.ID || pending[1].ID != a.ID {
		t.Fatalf("審査待ち（古い順・置き換えたものを除く）: %+v", pending)
	}

	if _, err := cards.FindByID(ctx, "missing"); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("ない学生証: err = %v, want ErrNotFound", err)
	}
	if got, err := cards.FindByID(ctx, old.ID); err != nil || got.SupersededAt == nil {
		t.Errorf("置き換えた学生証も探せる: got=%+v err=%v", got, err)
	}

	// 置き換えた学生証は審査できない
	t1 := t0.Add(3 * time.Hour)
	if err := cards.Review(ctx, old.ID, domain.StudentCardStatusApproved, admin.ID, t1); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("置き換えた学生証の承認: err = %v, want ErrConflict", err)
	}
	if err := cards.Review(ctx, "missing", domain.StudentCardStatusApproved, admin.ID, t1); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("ない学生証の承認: err = %v, want ErrConflict", err)
	}
	if err := cards.Review(ctx, a.ID, domain.StudentCardStatusApproved, admin.ID, t1); err != nil {
		t.Fatal(err)
	}
	got, err := cards.FindByID(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.StudentCardStatusApproved || got.ReviewedBy == nil || *got.ReviewedBy != admin.ID ||
		got.ReviewedAt == nil || !got.ReviewedAt.Equal(t1) {
		t.Errorf("承認後: %+v", got)
	}
	// 審査済みはもう一度審査できない
	if err := cards.Review(ctx, a.ID, domain.StudentCardStatusRejected, admin.ID, t1); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("承認済みの差し戻し: err = %v, want ErrConflict", err)
	}
	if pending, err := cards.ListPending(ctx); err != nil || len(pending) != 1 || pending[0].ID != b.ID {
		t.Errorf("承認後の審査待ち: %+v err=%v", pending, err)
	}

	// 閲覧記録は新しい順
	for _, at := range []time.Time{t0, t1, t0.Add(time.Minute)} {
		if err := views.Create(ctx, &domain.StudentCardView{StudentCardID: b.ID, AdminUserID: admin.ID, ViewedAt: at, CreatedAt: at, UpdatedAt: at}); err != nil {
			t.Fatal(err)
		}
	}
	vs, err := views.ListByCard(ctx, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 3 || !vs[0].ViewedAt.Equal(t1) || !vs[1].ViewedAt.Equal(t0.Add(time.Minute)) || !vs[2].ViewedAt.Equal(t0) ||
		vs[0].AdminUserID != admin.ID || vs[0].StudentCardID != b.ID || vs[0].ID == "" {
		t.Errorf("閲覧記録: %+v", vs)
	}
	if vs, err := views.ListByCard(ctx, a.ID); err != nil || len(vs) != 0 {
		t.Errorf("閲覧記録のない学生証: %+v err=%v", vs, err)
	}

	// ユーザーのまとめての取得
	m, err := users.FindByIDs(ctx, []string{us[0].ID, admin.ID, "missing"})
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 2 || m[admin.ID].StudentNumber != "26238003" || m[us[0].ID].StudentNumber != "26238001" {
		t.Errorf("FindByIDs: %+v", m)
	}
	if m, err := users.FindByIDs(ctx, nil); err != nil || len(m) != 0 {
		t.Errorf("FindByIDs(nil): %+v err=%v", m, err)
	}
}
