package service_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/clock"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/repository"
	"github.com/takuma0403/osagari_web/server/internal/service"
	"github.com/takuma0403/osagari_web/server/internal/storage"
	"github.com/takuma0403/osagari_web/server/internal/testutil/testdb"
)

const cardBucket = "student-cards"

type cardFixture struct {
	gdb     *gorm.DB
	svc     *service.StudentCardService
	users   *repository.UserRepository
	cards   *repository.StudentCardRepository
	storage *storage.Memory
	clock   *clock.Fake
	user    *domain.User
}

func newCardFixture(t *testing.T) *cardFixture {
	t.Helper()
	gdb := testdb.New(t)
	f := &cardFixture{
		gdb:     gdb,
		users:   repository.NewUserRepository(gdb),
		cards:   repository.NewStudentCardRepository(gdb),
		storage: storage.NewMemory(),
		clock:   clock.NewFake(t0),
	}
	f.svc = service.NewStudentCardService(service.StudentCardDeps{
		Tx:      repository.NewTransactor(gdb),
		Users:   f.users,
		Cards:   f.cards,
		Storage: f.storage,
		Bucket:  cardBucket,
		Clock:   f.clock,
	})
	f.user = f.createUser(t, email, "26238001")
	return f
}

func (f *cardFixture) createUser(t *testing.T, email, number string) *domain.User {
	t.Helper()
	u := &domain.User{
		Email: email, StudentNumber: number, AdmissionYear: 2026, DepartmentCode: "238",
		ExpiresOn: time.Date(2030, 3, 31, 0, 0, 0, 0, time.UTC),
		Status:    domain.UserStatusActive, SellerStatus: domain.SellerStatusNone,
		PasswordHash: "hash", CodeVerifiedAt: t0, CreatedAt: t0, UpdatedAt: t0,
	}
	if err := f.users.Create(context.Background(), u); err != nil {
		t.Fatal(err)
	}
	return u
}

// reload はセッションの検証と同じく、DB から本人を読み直す（リクエストごとに最新の状態になる）。
func (f *cardFixture) reload(t *testing.T) *domain.User {
	t.Helper()
	u, err := f.users.FindByID(context.Background(), f.user.ID)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

// upload は署名付きURLを発行し、ブラウザがその URL に PUT したことにする。
func (f *cardFixture) upload(t *testing.T, contentType string, size int64) string {
	t.Helper()
	up, err := f.svc.CreateUpload(context.Background(), f.reload(t), contentType)
	if err != nil {
		t.Fatal(err)
	}
	f.storage.Put(cardBucket, up.ObjectKey, storage.ObjectInfo{Size: size, ContentType: contentType})
	return up.ObjectKey
}

func (f *cardFixture) setSellerStatus(t *testing.T, s domain.SellerStatus) {
	t.Helper()
	all := []domain.SellerStatus{domain.SellerStatusNone, domain.SellerStatusPending, domain.SellerStatusApproved, domain.SellerStatusRejected}
	if err := f.users.UpdateSellerStatus(context.Background(), f.user.ID, all, s, t0); err != nil {
		t.Fatal(err)
	}
}

func TestStudentCardCreateUpload(t *testing.T) {
	ctx := context.Background()
	f := newCardFixture(t)

	up, err := f.svc.CreateUpload(ctx, f.user, "image/webp")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(up.ObjectKey, f.user.ID+"/") || up.Method != "PUT" ||
		up.Headers["Content-Type"] != "image/webp" || up.Headers["x-goog-content-length-range"] != "0,5242880" ||
		up.Headers["x-goog-if-generation-match"] != "0" || !up.ExpiresAt.Equal(t0.Add(15*time.Minute)) ||
		!strings.Contains(up.URL, "/"+cardBucket+"/"+up.ObjectKey) {
		t.Errorf("upload = %+v", up)
	}
	other, err := f.svc.CreateUpload(ctx, f.user, "image/jpeg")
	if err != nil {
		t.Fatal(err)
	}
	if other.ObjectKey == up.ObjectKey {
		t.Error("オブジェクト名が毎回変わらない")
	}

	if _, err := f.svc.CreateUpload(ctx, f.user, "image/png"); apperr.CodeOf(err) != apperr.CodeBadRequest {
		t.Errorf("PNG: err = %v, want bad_request", err)
	}
	f.setSellerStatus(t, domain.SellerStatusApproved)
	if _, err := f.svc.CreateUpload(ctx, f.reload(t), "image/jpeg"); apperr.CodeOf(err) != apperr.CodeInvalidStatusTransition {
		t.Errorf("承認済み: err = %v, want invalid_status_transition", err)
	}

	f.storage.Err = errors.New("storage down")
	f.setSellerStatus(t, domain.SellerStatusNone)
	if _, err := f.svc.CreateUpload(ctx, f.reload(t), "image/jpeg"); apperr.CodeOf(err) != apperr.CodeInternal {
		t.Errorf("ストレージの障害: err = %v, want internal_error", err)
	}
}

func TestStudentCardSubmit(t *testing.T) {
	ctx := context.Background()
	f := newCardFixture(t)

	// 未登録 → 審査中
	if st, err := f.svc.Current(ctx, f.user); err != nil || st.Status != domain.SellerStatusNone || st.SubmittedAt != nil {
		t.Fatalf("登録前: st=%+v err=%v", st, err)
	}
	key := f.upload(t, "image/jpeg", 300<<10)
	st, err := f.svc.Submit(ctx, f.reload(t), key)
	if err != nil {
		t.Fatal(err)
	}
	if st.Status != domain.SellerStatusPending || st.SubmittedAt == nil || !st.SubmittedAt.Equal(t0) || st.ReviewedAt != nil {
		t.Errorf("登録後: %+v", st)
	}
	u := f.reload(t)
	if u.SellerStatus != domain.SellerStatusPending {
		t.Errorf("seller_status = %s, want pending", u.SellerStatus)
	}
	if st, err := f.svc.Current(ctx, u); err != nil || st.Status != domain.SellerStatusPending || !st.SubmittedAt.Equal(t0) {
		t.Errorf("Current: st=%+v err=%v", st, err)
	}
	card, err := f.cards.FindCurrentByUser(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if card.ObjectKey != key || card.ContentType != "image/jpeg" || card.SizeBytes != 300<<10 || card.Status != domain.StudentCardStatusPending {
		t.Errorf("card = %+v", card)
	}
}

func TestStudentCardSubmitInvalid(t *testing.T) {
	ctx := context.Background()
	otherEmail := "26238002@edu.cc.saga-u.ac.jp"

	tests := []struct {
		name string
		// prepare はアップロードを用意して、登録に送るオブジェクト名を返す
		prepare func(t *testing.T, f *cardFixture) string
		code    apperr.Code
	}{
		{"ほかの人のオブジェクト", func(t *testing.T, f *cardFixture) string {
			other := f.createUser(t, otherEmail, "26238002")
			up, err := f.svc.CreateUpload(ctx, other, "image/jpeg")
			if err != nil {
				t.Fatal(err)
			}
			f.storage.Put(cardBucket, up.ObjectKey, storage.ObjectInfo{Size: 100, ContentType: "image/jpeg"})
			return up.ObjectKey
		}, service.CodeInvalidStudentCardUpload},
		{"発行していない形の名前", func(t *testing.T, f *cardFixture) string {
			key := f.user.ID + "/../x"
			f.storage.Put(cardBucket, key, storage.ObjectInfo{Size: 100, ContentType: "image/jpeg"})
			return key
		}, service.CodeInvalidStudentCardUpload},
		{"アップロードしていない", func(t *testing.T, f *cardFixture) string {
			up, err := f.svc.CreateUpload(ctx, f.user, "image/jpeg")
			if err != nil {
				t.Fatal(err)
			}
			return up.ObjectKey
		}, service.CodeInvalidStudentCardUpload},
		{"形式が違う", func(t *testing.T, f *cardFixture) string {
			up, err := f.svc.CreateUpload(ctx, f.user, "image/jpeg")
			if err != nil {
				t.Fatal(err)
			}
			f.storage.Put(cardBucket, up.ObjectKey, storage.ObjectInfo{Size: 100, ContentType: "image/png"})
			return up.ObjectKey
		}, service.CodeInvalidStudentCardUpload},
		{"空のファイル", func(t *testing.T, f *cardFixture) string {
			return f.upload(t, "image/jpeg", 0)
		}, service.CodeInvalidStudentCardUpload},
		{"5MB を超える", func(t *testing.T, f *cardFixture) string {
			return f.upload(t, "image/jpeg", 5<<20+1)
		}, service.CodeInvalidStudentCardUpload},
		{"登録済みのオブジェクト", func(t *testing.T, f *cardFixture) string {
			key := f.upload(t, "image/jpeg", 100)
			if _, err := f.svc.Submit(ctx, f.reload(t), key); err != nil {
				t.Fatal(err)
			}
			return key
		}, service.CodeInvalidStudentCardUpload},
		{"承認済み", func(t *testing.T, f *cardFixture) string {
			key := f.upload(t, "image/jpeg", 100)
			f.setSellerStatus(t, domain.SellerStatusApproved)
			return key
		}, apperr.CodeInvalidStatusTransition},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newCardFixture(t)
			key := tt.prepare(t, f)
			before := f.reload(t).SellerStatus
			if _, err := f.svc.Submit(ctx, f.reload(t), key); apperr.CodeOf(err) != tt.code {
				t.Fatalf("err = %v, want %s", err, tt.code)
			}
			if got := f.reload(t).SellerStatus; got != before {
				t.Errorf("seller_status が %s から %s に変わった", before, got)
			}
		})
	}

	t.Run("5MB ちょうどは受け付ける", func(t *testing.T) {
		f := newCardFixture(t)
		if _, err := f.svc.Submit(ctx, f.reload(t), f.upload(t, "image/webp", 5<<20)); err != nil {
			t.Fatal(err)
		}
	})
}

func TestStudentCardResubmit(t *testing.T) {
	ctx := context.Background()

	for _, from := range []domain.SellerStatus{domain.SellerStatusPending, domain.SellerStatusRejected} {
		t.Run(string(from), func(t *testing.T) {
			f := newCardFixture(t)
			first := f.upload(t, "image/jpeg", 100)
			if _, err := f.svc.Submit(ctx, f.reload(t), first); err != nil {
				t.Fatal(err)
			}
			f.setSellerStatus(t, from)

			f.clock.Advance(time.Hour)
			second := f.upload(t, "image/jpeg", 200)
			st, err := f.svc.Submit(ctx, f.reload(t), second)
			if err != nil {
				t.Fatal(err)
			}
			if st.Status != domain.SellerStatusPending || !st.SubmittedAt.Equal(t0.Add(time.Hour)) {
				t.Errorf("再提出後: %+v", st)
			}
			cur, err := f.cards.FindCurrentByUser(ctx, f.user.ID)
			if err != nil || cur.ObjectKey != second {
				t.Errorf("現在の学生証: cur=%+v err=%v", cur, err)
			}
			// 前の学生証も残す（ADR 0014）
			if ok, err := f.cards.ExistsByObjectKey(ctx, first); err != nil || !ok {
				t.Errorf("前の学生証が残っていない: ok=%v err=%v", ok, err)
			}
		})
	}
}

// 同時に2回登録しても、現在の学生証は1件だけになる（DB の二重処理の防止）。
func TestStudentCardSubmitConcurrent(t *testing.T) {
	ctx := context.Background()
	f := newCardFixture(t)
	keys := []string{f.upload(t, "image/jpeg", 100), f.upload(t, "image/jpeg", 100)}
	u := f.reload(t)

	var wg sync.WaitGroup
	errs := make([]error, len(keys))
	for i, key := range keys {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = f.svc.Submit(ctx, u, key)
		}()
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil && apperr.CodeOf(err) != apperr.CodeInvalidStatusTransition {
			t.Errorf("%d: err = %v", i, err)
		}
	}
	cur, err := f.cards.FindCurrentByUser(ctx, f.user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if cur.ObjectKey != keys[0] && cur.ObjectKey != keys[1] {
		t.Errorf("現在の学生証 = %+v", cur)
	}
	if f.reload(t).SellerStatus != domain.SellerStatusPending {
		t.Error("seller_status が pending でない")
	}
}
