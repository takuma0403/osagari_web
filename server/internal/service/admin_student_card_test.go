package service_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/repository"
	"github.com/takuma0403/osagari_web/server/internal/service"
)

const adminNumber = "26238099"

type adminCardFixture struct {
	*cardFixture
	admin    *service.AdminStudentCardService
	views    *repository.StudentCardViewRepository
	adminU   *domain.User
	outsider *domain.User
}

func newAdminCardFixture(t *testing.T) *adminCardFixture {
	t.Helper()
	cf := newCardFixture(t)
	gdb := cf.gdb
	admins, err := domain.ParseAdminList(adminNumber)
	if err != nil {
		t.Fatal(err)
	}
	f := &adminCardFixture{cardFixture: cf, views: repository.NewStudentCardViewRepository(gdb)}
	f.admin = service.NewAdminStudentCardService(service.AdminStudentCardDeps{
		Tx:      repository.NewTransactor(gdb),
		Users:   cf.users,
		Cards:   cf.cards,
		Views:   f.views,
		Storage: cf.storage,
		Bucket:  cardBucket,
		Admins:  admins,
		Clock:   cf.clock,
	})
	f.adminU = cf.createUser(t, adminNumber+"@edu.cc.saga-u.ac.jp", adminNumber)
	f.outsider = cf.createUser(t, "26238050@edu.cc.saga-u.ac.jp", "26238050")
	return f
}

// submit は本人が学生証を登録し、その学生証の ID を返す。
func (f *adminCardFixture) submit(t *testing.T) string {
	t.Helper()
	key := f.upload(t, "image/jpeg", 100)
	if _, err := f.svc.Submit(context.Background(), f.reload(t), key); err != nil {
		t.Fatal(err)
	}
	cur, err := f.cards.FindCurrentByUser(context.Background(), f.user.ID)
	if err != nil {
		t.Fatal(err)
	}
	return cur.ID
}

// 管理者でなければ、どの操作も存在しないパスと同じ 404 になる（ADR 0015）。
func TestAdminStudentCardRequiresAdmin(t *testing.T) {
	ctx := context.Background()
	f := newAdminCardFixture(t)
	id := f.submit(t)

	for _, u := range []*domain.User{nil, f.outsider, f.reload(t)} {
		calls := map[string]func() error{
			"RequireAdmin": func() error { return f.admin.RequireAdmin(u) },
			"ListPending":  func() error { _, err := f.admin.ListPending(ctx, u); return err },
			"CreateView":   func() error { _, err := f.admin.CreateView(ctx, u, id); return err },
			"ListViews":    func() error { _, err := f.admin.ListViews(ctx, u, id); return err },
			"Approve":      func() error { _, err := f.admin.Approve(ctx, u, id); return err },
			"Reject":       func() error { _, err := f.admin.Reject(ctx, u, id); return err },
		}
		for name, call := range calls {
			err := call()
			var ae *apperr.Error
			if !errors.As(err, &ae) || ae.Code != apperr.CodeNotFound || ae.Message != apperr.NotFound("").Message {
				t.Errorf("%s（%v）: err = %v, want 既定の not_found", name, u, err)
			}
		}
	}
	if err := f.admin.RequireAdmin(f.adminU); err != nil {
		t.Errorf("管理者: err = %v", err)
	}
	// 何も変わっていない
	if card, _ := f.cards.FindByID(ctx, id); card.Status != domain.StudentCardStatusPending {
		t.Errorf("学生証 = %+v", card)
	}
	if vs, _ := f.views.ListByCard(ctx, id); len(vs) != 0 {
		t.Errorf("閲覧記録 = %+v", vs)
	}
}

func TestAdminStudentCardListPending(t *testing.T) {
	ctx := context.Background()
	f := newAdminCardFixture(t)
	if list, err := f.admin.ListPending(ctx, f.adminU); err != nil || len(list) != 0 {
		t.Fatalf("登録前: %+v err=%v", list, err)
	}
	id := f.submit(t)
	list, err := f.admin.ListPending(ctx, f.adminU)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Card.ID != id || list[0].User.ID != f.user.ID || list[0].User.StudentNumber != "26238001" {
		t.Errorf("一覧 = %+v", list)
	}
}

func TestAdminStudentCardView(t *testing.T) {
	ctx := context.Background()
	f := newAdminCardFixture(t)
	id := f.submit(t)

	img, err := f.admin.CreateView(ctx, f.adminU, id)
	if err != nil {
		t.Fatal(err)
	}
	card, _ := f.cards.FindByID(ctx, id)
	if img.Method != "GET" || !strings.Contains(img.URL, "/"+cardBucket+"/"+card.ObjectKey) ||
		!img.ExpiresAt.Equal(t0.Add(15*time.Minute)) || !img.ViewedAt.Equal(t0) {
		t.Errorf("閲覧URL = %+v", img)
	}
	f.clock.Advance(time.Minute)
	if _, err := f.admin.CreateView(ctx, f.adminU, id); err != nil {
		t.Fatal(err)
	}

	views, err := f.admin.ListViews(ctx, f.adminU, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 2 || !views[0].ViewedAt.Equal(t0.Add(time.Minute)) || !views[1].ViewedAt.Equal(t0) ||
		views[0].AdminStudentNumber != adminNumber {
		t.Errorf("閲覧記録 = %+v", views)
	}

	if _, err := f.admin.CreateView(ctx, f.adminU, "missing"); apperr.CodeOf(err) != apperr.CodeNotFound {
		t.Errorf("ない学生証の閲覧: err = %v", err)
	}
	if _, err := f.admin.ListViews(ctx, f.adminU, "missing"); apperr.CodeOf(err) != apperr.CodeNotFound {
		t.Errorf("ない学生証の閲覧記録: err = %v", err)
	}

	// ストレージの障害は 500。記録は残る（記録のない閲覧URLは作らない）
	f.storage.Err = errors.New("障害")
	if _, err := f.admin.CreateView(ctx, f.adminU, id); apperr.CodeOf(err) != apperr.CodeInternal {
		t.Errorf("ストレージの障害: err = %v", err)
	}
}

func TestAdminStudentCardViewImageDeleted(t *testing.T) {
	ctx := context.Background()
	f := newAdminCardFixture(t)
	deleted := t0
	card := &domain.StudentCard{
		UserID: f.user.ID, ObjectKey: f.user.ID + "/deleted", ContentType: "image/jpeg", SizeBytes: 100,
		Status: domain.StudentCardStatusApproved, SubmittedAt: t0, ImageDeletedAt: &deleted, CreatedAt: t0, UpdatedAt: t0,
	}
	if err := f.cards.Create(ctx, card); err != nil {
		t.Fatal(err)
	}
	if _, err := f.admin.CreateView(ctx, f.adminU, card.ID); apperr.CodeOf(err) != apperr.CodeNotFound {
		t.Errorf("画像を削除した学生証の閲覧: err = %v", err)
	}
	if vs, _ := f.views.ListByCard(ctx, card.ID); len(vs) != 0 {
		t.Errorf("閲覧記録 = %+v", vs)
	}
}

func TestAdminStudentCardReview(t *testing.T) {
	tests := []struct {
		name       string
		review     func(*service.AdminStudentCardService, context.Context, *domain.User, string) (service.AdminStudentCard, error)
		wantCard   domain.StudentCardStatus
		wantSeller domain.SellerStatus
	}{
		{"承認", (*service.AdminStudentCardService).Approve, domain.StudentCardStatusApproved, domain.SellerStatusApproved},
		{"差し戻し", (*service.AdminStudentCardService).Reject, domain.StudentCardStatusRejected, domain.SellerStatusRejected},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			f := newAdminCardFixture(t)
			id := f.submit(t)
			f.clock.Advance(time.Hour)

			got, err := tt.review(f.admin, ctx, f.adminU, id)
			if err != nil {
				t.Fatal(err)
			}
			if got.Card.Status != tt.wantCard || got.Card.ReviewedBy == nil || *got.Card.ReviewedBy != f.adminU.ID ||
				got.Card.ReviewedAt == nil || !got.Card.ReviewedAt.Equal(t0.Add(time.Hour)) ||
				got.User.ID != f.user.ID || got.User.SellerStatus != tt.wantSeller {
				t.Errorf("審査後 = card %+v user %+v", got.Card, got.User)
			}
			if f.reload(t).SellerStatus != tt.wantSeller {
				t.Error("本人の承認状態が変わっていない")
			}

			// 審査済みはもう一度審査できない
			for _, again := range tests {
				if _, err := again.review(f.admin, ctx, f.adminU, id); apperr.CodeOf(err) != apperr.CodeInvalidStatusTransition {
					t.Errorf("審査済みの%s: err = %v", again.name, err)
				}
			}
			if _, err := tt.review(f.admin, ctx, f.adminU, "missing"); apperr.CodeOf(err) != apperr.CodeNotFound {
				t.Errorf("ない学生証: err = %v", err)
			}
		})
	}
}

// 再提出で置き換えられた学生証は審査できず、本人の承認状態も変わらない。
func TestAdminStudentCardReviewSuperseded(t *testing.T) {
	ctx := context.Background()
	f := newAdminCardFixture(t)
	old := f.submit(t)
	cur := f.submit(t)
	if old == cur {
		t.Fatal("再提出で学生証が変わっていない")
	}
	if _, err := f.admin.Approve(ctx, f.adminU, old); apperr.CodeOf(err) != apperr.CodeInvalidStatusTransition {
		t.Errorf("置き換えた学生証の承認: err = %v", err)
	}
	if f.reload(t).SellerStatus != domain.SellerStatusPending {
		t.Error("本人の承認状態が変わった")
	}
	if _, err := f.admin.Approve(ctx, f.adminU, cur); err != nil {
		t.Errorf("現在の学生証の承認: err = %v", err)
	}
}

// 本人の承認状態が審査中でなければ、学生証の更新も取り消す（トランザクション）。
func TestAdminStudentCardReviewSellerNotPending(t *testing.T) {
	ctx := context.Background()
	f := newAdminCardFixture(t)
	id := f.submit(t)
	f.setSellerStatus(t, domain.SellerStatusNone)
	if _, err := f.admin.Approve(ctx, f.adminU, id); apperr.CodeOf(err) != apperr.CodeInvalidStatusTransition {
		t.Errorf("err = %v", err)
	}
	if card, _ := f.cards.FindByID(ctx, id); card.Status != domain.StudentCardStatusPending || card.ReviewedAt != nil {
		t.Errorf("学生証が更新された: %+v", card)
	}
}

// 同時に承認と差し戻しをしても、片方だけが成功する。
func TestAdminStudentCardReviewConcurrent(t *testing.T) {
	ctx := context.Background()
	f := newAdminCardFixture(t)
	id := f.submit(t)

	calls := []func() (service.AdminStudentCard, error){
		func() (service.AdminStudentCard, error) { return f.admin.Approve(ctx, f.adminU, id) },
		func() (service.AdminStudentCard, error) { return f.admin.Reject(ctx, f.adminU, id) },
	}
	var wg sync.WaitGroup
	errs := make([]error, len(calls))
	for i, call := range calls {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = call()
		}()
	}
	wg.Wait()

	ok := 0
	for i, err := range errs {
		switch {
		case err == nil:
			ok++
		case apperr.CodeOf(err) != apperr.CodeInvalidStatusTransition:
			t.Errorf("%d: err = %v", i, err)
		}
	}
	if ok != 1 {
		t.Fatalf("成功した数 = %d, want 1（%v）", ok, errs)
	}
	card, _ := f.cards.FindByID(ctx, id)
	want := map[domain.StudentCardStatus]domain.SellerStatus{
		domain.StudentCardStatusApproved: domain.SellerStatusApproved,
		domain.StudentCardStatusRejected: domain.SellerStatusRejected,
	}
	if s, found := want[card.Status]; !found || f.reload(t).SellerStatus != s {
		t.Errorf("学生証 %s と本人 %s が一致しない", card.Status, f.reload(t).SellerStatus)
	}
}
