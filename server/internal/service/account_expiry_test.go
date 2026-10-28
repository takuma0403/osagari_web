package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/payment"
	"github.com/takuma0403/osagari_web/server/internal/repository"
	"github.com/takuma0403/osagari_web/server/internal/service"
	"github.com/takuma0403/osagari_web/server/internal/storage"
)

// expiryFixture は取引のテストの DB・時計に、アカウントの失効と学生証画像の削除（ADR 0033）を足す。
// 利用者の有効期限はすべて 2030-03-31。
type expiryFixture struct {
	*dealFixture
	expiry   *service.AccountExpiryService
	cards    *repository.StudentCardRepository
	sessions *repository.SessionRepository
}

func newExpiryFixture(t *testing.T) *expiryFixture {
	t.Helper()
	f := &expiryFixture{dealFixture: newDealFixture(t)}
	f.cards = repository.NewStudentCardRepository(f.gdb)
	f.sessions = repository.NewSessionRepository(f.gdb)
	f.expiry = service.NewAccountExpiryService(service.AccountExpiryDeps{
		Tx:           repository.NewTransactor(f.gdb),
		Users:        f.users,
		StudentCards: f.cards,
		Items:        f.items,
		Transactions: f.txs,
		Sessions:     f.sessions,
		Payments:     payment.NewMock(f.payments, f.clock),
		Storage:      f.storage,
		Bucket:       cardBucket,
		Clock:        f.clock,
	})
	return f
}

// 日本時間の日付の始まり（UTC では前日の15時）。
func jstMidnight(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC).Add(-9 * time.Hour)
}

// extend は u の有効期限を1年延ばし、失効の対象から外す。
func (f *expiryFixture) extend(t *testing.T, users ...*domain.User) {
	t.Helper()
	for _, u := range users {
		if err := f.gdb.Exec("UPDATE users SET expires_on = '2031-03-31' WHERE id = ?", u.ID).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func (f *expiryFixture) expireAccounts(t *testing.T, want int) {
	t.Helper()
	if n, err := f.expiry.ExpireAccounts(context.Background()); err != nil || n != want {
		t.Fatalf("失効させた人数: n=%d err=%v want %d", n, err, want)
	}
}

func (f *expiryFixture) userStatus(t *testing.T, u *domain.User) domain.UserStatus {
	t.Helper()
	got, err := f.users.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	return got.Status
}

func (f *expiryFixture) itemStatusOf(t *testing.T, id string) domain.ItemStatus {
	t.Helper()
	it, err := f.items.FindByID(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return it.Status
}

// addCard は u の学生証を登録したことにする（画像もバケットに置く）。
func (f *expiryFixture) addCard(t *testing.T, u *domain.User, key string) *domain.StudentCard {
	t.Helper()
	now := f.clock.Now()
	c := &domain.StudentCard{
		UserID: u.ID, ObjectKey: u.ID + "/" + key, ContentType: "image/jpeg", SizeBytes: 1024,
		Status: domain.StudentCardStatusPending, SubmittedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := f.cards.Create(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	f.storage.PutAt(cardBucket, c.ObjectKey, storage.ObjectInfo{Size: 1024, ContentType: "image/jpeg"}, now)
	return c
}

// 有効期限の日の終わり（日本時間）までは使え、翌日の日次ジョブで失効する。何度呼んでも結果は同じ（AUTH-3, ADR 0033）。
func TestExpireAccounts_有効期限の翌日に失効する(t *testing.T) {
	f := newExpiryFixture(t)
	f.extend(t, f.buyer, f.buyer2, f.stranger)

	f.advanceTo(jstMidnight(2030, 4, 1).Add(-time.Second))
	f.expireAccounts(t, 0)
	if got := f.userStatus(t, f.seller); got != domain.UserStatusActive {
		t.Fatalf("有効期限の日: status = %s, want active", got)
	}

	f.advanceTo(jstMidnight(2030, 4, 1))
	f.expireAccounts(t, 1)
	if got := f.userStatus(t, f.seller); got != domain.UserStatusExpired {
		t.Errorf("翌日: status = %s, want expired", got)
	}
	if got := f.userStatus(t, f.buyer); got != domain.UserStatusActive {
		t.Errorf("期限が先の人: status = %s, want active", got)
	}
	f.expireAccounts(t, 0)
}

// 停止中のアカウントも有効期限で失効する（学生証の保持期間を始めるため）。
func TestExpireAccounts_停止中も失効する(t *testing.T) {
	f := newExpiryFixture(t)
	f.extend(t, f.seller, f.buyer2, f.stranger)
	if err := f.gdb.Exec("UPDATE users SET status = 'suspended' WHERE id = ?", f.buyer.ID).Error; err != nil {
		t.Fatal(err)
	}
	f.advanceTo(jstMidnight(2030, 4, 1))
	f.expireAccounts(t, 1)
	if got := f.userStatus(t, f.buyer); got != domain.UserStatusExpired {
		t.Errorf("status = %s, want expired", got)
	}
}

// 出品者が失効すると、商品を取り下げ、購入希望を期限切れにし、取引中の取引をキャンセルして返金する。
// 学生証はすべての行に失効日 + 3年の削除予定日を入れ、セッションを無効にする。完了した取引は変えない（ADR 0033）。
func TestExpireAccounts_出品者の失効(t *testing.T) {
	ctx := context.Background()
	f := newExpiryFixture(t)
	f.extend(t, f.buyer, f.buyer2, f.stranger)

	completed := f.dealOn(t, f.buyer2)
	f.completeDeal(t, f.buyer2, completed.ID)
	inDeal := f.approved(t) // f.item を f.buyer と取引中
	requested := f.publish(t)
	req, err := f.deals.CreateRequest(ctx, f.buyer2, requested.ShareID, domain.PaymentMethodCard)
	if err != nil {
		t.Fatal(err)
	}
	published := f.publish(t)
	old := f.addCard(t, f.seller, "old")
	if _, err := f.cards.SupersedeCurrent(ctx, f.seller.ID, f.clock.Now()); err != nil {
		t.Fatal(err)
	}
	cur := f.addCard(t, f.seller, "cur")
	session := &domain.Session{UserID: f.seller.ID, TokenHash: "seller-session", ExpiresAt: jstMidnight(2031, 1, 1), CreatedAt: t0, UpdatedAt: t0}
	if err := f.sessions.Create(ctx, session); err != nil {
		t.Fatal(err)
	}

	f.advanceTo(jstMidnight(2030, 4, 1))
	f.expireAccounts(t, 1)

	if got := f.txStatus(t, inDeal.ID); got != domain.TransactionStatusCancelled {
		t.Errorf("取引中: status = %s, want cancelled", got)
	}
	tr, err := f.txs.FindByID(ctx, inDeal.ID)
	if err != nil {
		t.Fatal(err)
	}
	if tr.BuyerCancelAgreedAt != nil || tr.SellerCancelAgreedAt != nil {
		t.Errorf("失効によるキャンセルに同意日時が入った: %+v", tr)
	}
	wantKinds(t, f.paymentKinds(t, inDeal.ID), domain.PaymentKindAuthorize, domain.PaymentKindCapture, domain.PaymentKindRefund)
	if got := f.txStatus(t, req.ID); got != domain.TransactionStatusExpired {
		t.Errorf("購入希望: status = %s, want expired", got)
	}
	wantKinds(t, f.paymentKinds(t, req.ID), domain.PaymentKindAuthorize, domain.PaymentKindRelease)
	for name, id := range map[string]string{"取引中だった商品": f.item.ID, "購入希望ありの商品": requested.ID, "公開中の商品": published.ID} {
		if got := f.itemStatusOf(t, id); got != domain.ItemStatusWithdrawn {
			t.Errorf("%s: status = %s, want withdrawn", name, got)
		}
	}
	if got := f.txStatus(t, completed.ID); got != domain.TransactionStatusCompleted {
		t.Errorf("完了した取引: status = %s, want completed", got)
	}
	if got := f.itemStatusOf(t, completed.Item.ID); got != domain.ItemStatusCompleted {
		t.Errorf("完了した商品: status = %s, want completed", got)
	}
	wantDeleteAfter := time.Date(2033, 4, 1, 0, 0, 0, 0, time.UTC)
	for _, c := range []*domain.StudentCard{old, cur} {
		got, err := f.cards.FindByID(ctx, c.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.DeleteAfter == nil || !got.DeleteAfter.Equal(wantDeleteAfter) {
			t.Errorf("削除予定日 = %v, want %v", got.DeleteAfter, wantDeleteAfter)
		}
	}
	if _, err := f.sessions.FindActiveByTokenHash(ctx, "seller-session", f.clock.Now()); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("セッションが残っている: err = %v", err)
	}
	for _, u := range []*domain.User{f.buyer, f.buyer2} {
		if got := f.userStatus(t, u); got != domain.UserStatusActive {
			t.Errorf("相手のアカウント: status = %s, want active", got)
		}
	}

	// 2回目は何もしない（返金・解放を二重に記録しない）
	f.expireAccounts(t, 0)
	wantKinds(t, f.paymentKinds(t, inDeal.ID), domain.PaymentKindAuthorize, domain.PaymentKindCapture, domain.PaymentKindRefund)
	wantKinds(t, f.paymentKinds(t, req.ID), domain.PaymentKindAuthorize, domain.PaymentKindRelease)
}

// 購入者が失効すると、購入希望を期限切れにし、取引中の取引をキャンセルして返金する。商品は公開中に戻し、出品者の商品は取り下げない（ADR 0033）。
func TestExpireAccounts_購入者の失効(t *testing.T) {
	ctx := context.Background()
	f := newExpiryFixture(t)
	f.extend(t, f.seller, f.buyer2, f.stranger)

	inDeal := f.approved(t) // f.item を f.buyer と取引中
	requested := f.publish(t)
	req, err := f.deals.CreateRequest(ctx, f.buyer, requested.ShareID, domain.PaymentMethodCard)
	if err != nil {
		t.Fatal(err)
	}
	// ほかの人の購入希望が残る商品は購入希望ありのまま
	shared := f.publish(t)
	reqShared, err := f.deals.CreateRequest(ctx, f.buyer, shared.ShareID, domain.PaymentMethodCard)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.deals.CreateRequest(ctx, f.buyer2, shared.ShareID, domain.PaymentMethodCard); err != nil {
		t.Fatal(err)
	}

	f.advanceTo(jstMidnight(2030, 4, 1))
	f.expireAccounts(t, 1)

	if got := f.txStatus(t, inDeal.ID); got != domain.TransactionStatusCancelled {
		t.Errorf("取引中: status = %s, want cancelled", got)
	}
	wantKinds(t, f.paymentKinds(t, inDeal.ID), domain.PaymentKindAuthorize, domain.PaymentKindCapture, domain.PaymentKindRefund)
	for _, id := range []string{req.ID, reqShared.ID} {
		if got := f.txStatus(t, id); got != domain.TransactionStatusExpired {
			t.Errorf("購入希望: status = %s, want expired", got)
		}
	}
	for name, tt := range map[string]struct {
		id   string
		want domain.ItemStatus
	}{
		"取引中だった商品":     {f.item.ID, domain.ItemStatusPublished},
		"購入希望がなくなった商品": {requested.ID, domain.ItemStatusPublished},
		"ほかの購入希望が残る商品": {shared.ID, domain.ItemStatusRequested},
	} {
		if got := f.itemStatusOf(t, tt.id); got != tt.want {
			t.Errorf("%s: status = %s, want %s", name, got, tt.want)
		}
	}
	if got := f.userStatus(t, f.seller); got != domain.UserStatusActive {
		t.Errorf("出品者: status = %s, want active", got)
	}
}

// 削除予定日（日本時間）の当日に画像を消して日時を記録する。画像がすでになくても記録し、行は消さない（CARD, JOBS-1）。
func TestDeleteStudentCardImages(t *testing.T) {
	ctx := context.Background()
	f := newExpiryFixture(t)
	f.extend(t, f.buyer, f.buyer2, f.stranger)
	cur := f.addCard(t, f.seller, "cur")
	missing := f.addCard(t, f.buyer, "missing")
	if err := f.storage.Delete(ctx, cardBucket, missing.ObjectKey); err != nil {
		t.Fatal(err)
	}
	if err := f.gdb.Exec("UPDATE student_cards SET delete_after = '2033-04-01' WHERE id = ?", missing.ID).Error; err != nil {
		t.Fatal(err)
	}
	f.advanceTo(jstMidnight(2030, 4, 1))
	f.expireAccounts(t, 1)

	f.advanceTo(jstMidnight(2033, 4, 1).Add(-time.Second))
	if n, err := f.expiry.DeleteStudentCardImages(ctx); err != nil || n != 0 {
		t.Fatalf("削除予定日の前日: n=%d err=%v", n, err)
	}
	if !f.storage.Has(cardBucket, cur.ObjectKey) {
		t.Fatal("削除予定日の前に画像を消した")
	}

	f.advanceTo(jstMidnight(2033, 4, 1))
	if n, err := f.expiry.DeleteStudentCardImages(ctx); err != nil || n != 2 {
		t.Fatalf("削除予定日: n=%d err=%v, want 2", n, err)
	}
	if f.storage.Has(cardBucket, cur.ObjectKey) {
		t.Error("画像が残っている")
	}
	for _, c := range []*domain.StudentCard{cur, missing} {
		got, err := f.cards.FindByID(ctx, c.ID)
		if err != nil {
			t.Fatalf("行が消えた: %v", err)
		}
		if got.ImageDeletedAt == nil || !got.ImageDeletedAt.Equal(f.clock.Now()) {
			t.Errorf("画像を消した日時 = %v", got.ImageDeletedAt)
		}
	}
	if n, err := f.expiry.DeleteStudentCardImages(ctx); err != nil || n != 0 {
		t.Errorf("2回目: n=%d err=%v, want 0", n, err)
	}
}

// ストレージの障害で消せなかった画像は記録せず、次の実行で消す。
func TestDeleteStudentCardImages_ストレージの障害(t *testing.T) {
	ctx := context.Background()
	f := newExpiryFixture(t)
	f.extend(t, f.buyer, f.buyer2, f.stranger)
	c := f.addCard(t, f.seller, "cur")
	f.advanceTo(jstMidnight(2030, 4, 1))
	f.expireAccounts(t, 1)
	f.advanceTo(jstMidnight(2033, 4, 1))

	f.storage.Err = errors.New("ストレージの障害")
	if _, err := f.expiry.DeleteStudentCardImages(ctx); err == nil {
		t.Fatal("err = nil, want ストレージの障害")
	}
	if got, _ := f.cards.FindByID(ctx, c.ID); got.ImageDeletedAt != nil {
		t.Fatal("消せなかったのに記録した")
	}
	f.storage.Err = nil
	if n, err := f.expiry.DeleteStudentCardImages(ctx); err != nil || n != 1 {
		t.Errorf("再実行: n=%d err=%v, want 1", n, err)
	}
}

// 登録されないまま24時間たった学生証のオブジェクトだけを消す（STORAGE, ADR 0033）。
func TestDeleteUnregisteredStudentCardImages(t *testing.T) {
	ctx := context.Background()
	f := newExpiryFixture(t)
	registered := f.addCard(t, f.seller, "registered")
	info := storage.ObjectInfo{Size: 1024, ContentType: "image/jpeg"}
	oldKey := f.buyer.ID + "/old"
	newKey := f.buyer.ID + "/new"
	f.storage.PutAt(cardBucket, oldKey, info, f.clock.Now())
	f.advanceTo(f.clock.Now().Add(domain.UnregisteredUploadRetention - time.Minute))
	f.storage.PutAt(cardBucket, newKey, info, f.clock.Now())
	f.advanceTo(registered.CreatedAt.Add(domain.UnregisteredUploadRetention))

	if n, err := f.expiry.DeleteUnregisteredStudentCardImages(ctx); err != nil || n != 1 {
		t.Fatalf("n=%d err=%v, want 1", n, err)
	}
	for key, want := range map[string]bool{registered.ObjectKey: true, oldKey: false, newKey: true} {
		if got := f.storage.Has(cardBucket, key); got != want {
			t.Errorf("%s が残っているか = %v, want %v", key, got, want)
		}
	}
	if n, err := f.expiry.DeleteUnregisteredStudentCardImages(ctx); err != nil || n != 0 {
		t.Errorf("2回目: n=%d err=%v, want 0", n, err)
	}
}
