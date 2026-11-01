package service_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/payment"
	"github.com/takuma0403/osagari_web/server/internal/repository"
	"github.com/takuma0403/osagari_web/server/internal/service"
)

// adminFixture は取引・失効のテストの DB・時計に、管理者と管理画面の service（ADMIN, ADR 0034）を足す。
type adminFixture struct {
	*expiryFixture
	adminUsers   *service.AdminUserService
	adminDeals   *service.AdminDealService
	adminSetting *service.SettingService
	admin        *domain.User
}

func newAdminFixture(t *testing.T) *adminFixture {
	t.Helper()
	f := &adminFixture{expiryFixture: newExpiryFixture(t)}
	admins, err := domain.ParseAdminList(adminNumber)
	if err != nil {
		t.Fatal(err)
	}
	tx := repository.NewTransactor(f.gdb)
	gw := payment.NewMock(f.payments, f.clock)
	f.adminUsers = service.NewAdminUserService(service.AdminUserDeps{
		Tx: tx, Users: f.users, StudentCards: f.cards, Items: f.items, Transactions: f.txs,
		Sessions: f.sessions, Payments: gw, Admins: admins, Clock: f.clock,
	})
	f.adminDeals = service.NewAdminDealService(service.AdminDealDeps{
		Tx: tx, Users: f.users, Items: f.items, Transactions: f.txs, Payments: gw,
		Storage: f.storage, Bucket: itemBucket, Admins: admins, Clock: f.clock,
	})
	settings := repository.NewSettingRepository(f.gdb)
	f.adminSetting = service.NewSettingService(service.SettingDeps{
		Settings: settings, AdminSettings: settings, Users: f.users, Admins: admins, Clock: f.clock,
	})
	f.admin = f.createUser(t, adminNumber+"@edu.cc.saga-u.ac.jp", adminNumber, domain.SellerStatusNone)
	return f
}

func (f *adminFixture) addSession(t *testing.T, u *domain.User, hash string) {
	t.Helper()
	s := &domain.Session{UserID: u.ID, TokenHash: hash, ExpiresAt: f.clock.Now().Add(24 * time.Hour), CreatedAt: f.clock.Now(), UpdatedAt: f.clock.Now()}
	if err := f.sessions.Create(context.Background(), s); err != nil {
		t.Fatal(err)
	}
}

func (f *adminFixture) sessionActive(t *testing.T, hash string) bool {
	t.Helper()
	_, err := f.sessions.FindActiveByTokenHash(context.Background(), hash, f.clock.Now())
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		t.Fatal(err)
	}
	return err == nil
}

// 管理者でなければ、どの操作も存在しないパスと同じ 404 にする（ADR 0015）。
func TestAdmin_管理者以外は404(t *testing.T) {
	ctx := context.Background()
	f := newAdminFixture(t)
	tr := f.approved(t)
	u := f.stranger
	calls := map[string]func() error{
		"ユーザーの一覧": func() error { _, err := f.adminUsers.List(ctx, u, service.AdminUserListQuery{}); return err },
		"ユーザーの詳細": func() error { _, err := f.adminUsers.Get(ctx, u, f.buyer.ID); return err },
		"停止":      func() error { _, err := f.adminUsers.Suspend(ctx, u, f.buyer.ID); return err },
		"停止の解除":   func() error { _, err := f.adminUsers.Unsuspend(ctx, u, f.buyer.ID); return err },
		"有効期限の変更": func() error {
			_, err := f.adminUsers.UpdateExpiresOn(ctx, u, f.buyer.ID, time.Date(2031, 3, 31, 0, 0, 0, 0, time.UTC))
			return err
		},
		"削除":         func() error { _, err := f.adminUsers.Delete(ctx, u, f.buyer.ID); return err },
		"出品の一覧":      func() error { _, err := f.adminDeals.ListItems(ctx, u, service.AdminPageQuery{}); return err },
		"出品の停止":      func() error { _, err := f.adminDeals.SuspendItem(ctx, u, f.item.ID); return err },
		"取引の一覧":      func() error { _, err := f.adminDeals.ListTransactions(ctx, u, service.AdminPageQuery{}); return err },
		"取引のキャンセル":   func() error { _, err := f.adminDeals.CancelTransaction(ctx, u, tr.ID); return err },
		"手数料率":       func() error { _, err := f.adminSetting.AdminFeeRate(ctx, u); return err },
		"手数料率の変更":    func() error { _, err := f.adminSetting.UpdateFeeRate(ctx, u, 800); return err },
		"未ログイン（nil）": func() error { _, err := f.adminUsers.List(ctx, nil, service.AdminUserListQuery{}); return err },
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) { wantCode(t, call(), apperr.CodeNotFound) })
	}
	// 何も変わっていない
	if got := f.userStatus(t, f.buyer); got != domain.UserStatusActive {
		t.Errorf("status = %s", got)
	}
	if got := f.txStatus(t, tr.ID); got != domain.TransactionStatusInDeal {
		t.Errorf("取引 = %s", got)
	}
}

// ユーザーの一覧：絞り込み・検索・続きの位置、キャンセル回数・取引中の件数・退学申請の件数（ADMIN, ADR 0034）。
func TestAdminUserList(t *testing.T) {
	ctx := context.Background()
	f := newAdminFixture(t)

	// 双方の同意でキャンセルした取引は、購入者・出品者の両方に数える
	cancelled := f.dealOn(t, f.buyer2)
	if _, err := f.deals.Cancel(ctx, f.buyer2, cancelled.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.deals.Cancel(ctx, f.seller, cancelled.ID); err != nil {
		t.Fatal(err)
	}
	f.approved(t) // f.buyer と取引中
	if err := f.gdb.Exec("UPDATE users SET withdrawal_requested_at = ? WHERE id = ?", "2026-10-15T00:00:00.000000Z", f.stranger.ID).Error; err != nil {
		t.Fatal(err)
	}

	page, err := f.adminUsers.List(ctx, f.admin, service.AdminUserListQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Entries) != 5 || page.NextCursor != nil || page.WithdrawalRequestCount != 1 {
		t.Fatalf("一覧: %d 件 next=%v 退学申請=%d", len(page.Entries), page.NextCursor, page.WithdrawalRequestCount)
	}
	byID := map[string]*service.AdminUserView{}
	for _, v := range page.Entries {
		byID[v.ID] = v
	}
	for _, c := range []struct {
		u              *domain.User
		cancels, deals int
	}{{f.seller, 1, 1}, {f.buyer, 0, 1}, {f.buyer2, 1, 0}, {f.stranger, 0, 0}} {
		v := byID[c.u.ID]
		if v == nil || v.CancelCount != c.cancels || v.ActiveTransactionCount != c.deals {
			t.Errorf("%s: %+v, want cancels=%d deals=%d", c.u.StudentNumber, v, c.cancels, c.deals)
		}
	}

	filter := domain.AdminUserFilterWithdrawalRequested
	page, err = f.adminUsers.List(ctx, f.admin, service.AdminUserListQuery{Filter: &filter})
	if err != nil || len(page.Entries) != 1 || page.Entries[0].ID != f.stranger.ID {
		t.Fatalf("退学申請の絞り込み: %+v err=%v", page, err)
	}
	q := "26141"
	page, err = f.adminUsers.List(ctx, f.admin, service.AdminUserListQuery{Q: &q})
	if err != nil || len(page.Entries) != 2 {
		t.Fatalf("学籍番号の検索: %+v err=%v", page, err)
	}

	// 2件ずつ読むと、すべてを重ならずに読める
	seen := map[string]bool{}
	var cursor *string
	for i := 0; ; i++ {
		page, err := f.adminUsers.List(ctx, f.admin, service.AdminUserListQuery{AdminPageQuery: service.AdminPageQuery{Cursor: cursor, Limit: ptr(2)}})
		if err != nil {
			t.Fatal(err)
		}
		for _, v := range page.Entries {
			if seen[v.ID] {
				t.Fatalf("重複: %s", v.StudentNumber)
			}
			seen[v.ID] = true
		}
		if cursor = page.NextCursor; cursor == nil || i > 5 {
			break
		}
	}
	if len(seen) != 5 {
		t.Errorf("続きを読んだ件数 = %d, want 5", len(seen))
	}

	bad := domain.AdminUserFilter("deleted")
	for name, in := range map[string]service.AdminUserListQuery{
		"数字でない検索語":   {Q: ptr("abc")},
		"9桁の検索語":     {Q: ptr("123456789")},
		"不明な絞り込み":    {Filter: &bad},
		"件数が多すぎる":    {AdminPageQuery: service.AdminPageQuery{Limit: ptr(51)}},
		"壊れた cursor": {AdminPageQuery: service.AdminPageQuery{Cursor: ptr("broken")}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := f.adminUsers.List(ctx, f.admin, in)
			wantCode(t, err, apperr.CodeBadRequest)
		})
	}
}

// 停止はセッションを無効にするだけで、商品・取引はそのまま。解除すると有効に戻る（ADR 0034）。
func TestAdminUserSuspend(t *testing.T) {
	ctx := context.Background()
	f := newAdminFixture(t)
	tr := f.approved(t)
	f.addSession(t, f.seller, "seller-session")

	v, err := f.adminUsers.Suspend(ctx, f.admin, f.seller.ID)
	if err != nil {
		t.Fatal(err)
	}
	if v.Status != domain.UserStatusSuspended || v.ActiveTransactionCount != 1 {
		t.Errorf("停止後: %+v", v)
	}
	if f.sessionActive(t, "seller-session") {
		t.Error("停止してもセッションが残っている")
	}
	if got := f.txStatus(t, tr.ID); got != domain.TransactionStatusInDeal {
		t.Errorf("取引 = %s, want in_deal", got)
	}
	if got := f.itemStatus(t); got != domain.ItemStatusInDeal {
		t.Errorf("商品 = %s, want in_deal", got)
	}
	_, err = f.adminUsers.Suspend(ctx, f.admin, f.seller.ID)
	wantCode(t, err, apperr.CodeInvalidStatusTransition)

	v, err = f.adminUsers.Unsuspend(ctx, f.admin, f.seller.ID)
	if err != nil || v.Status != domain.UserStatusActive {
		t.Fatalf("解除: %+v err=%v", v, err)
	}
	_, err = f.adminUsers.Unsuspend(ctx, f.admin, f.seller.ID)
	wantCode(t, err, apperr.CodeInvalidStatusTransition)

	_, err = f.adminUsers.Suspend(ctx, f.admin, "missing")
	wantCode(t, err, apperr.CodeNotFound)
}

// 有効期限は有効・停止のユーザーだけ、今日（日本時間）以降の日付に変えられる（AUTH-3, ADR 0034）。
func TestAdminUserUpdateExpiresOn(t *testing.T) {
	ctx := context.Background()
	f := newAdminFixture(t)
	// t0 は 2026-10-14 18:00（日本時間）
	today := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)

	v, err := f.adminUsers.UpdateExpiresOn(ctx, f.admin, f.buyer.ID, today)
	if err != nil || !v.ExpiresOn.Equal(today) {
		t.Fatalf("今日にする: %+v err=%v", v, err)
	}
	_, err = f.adminUsers.UpdateExpiresOn(ctx, f.admin, f.buyer.ID, today.AddDate(0, 0, -1))
	wantCode(t, err, apperr.CodeBadRequest)

	if _, err := f.adminUsers.Suspend(ctx, f.admin, f.buyer.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.adminUsers.UpdateExpiresOn(ctx, f.admin, f.buyer.ID, today.AddDate(1, 0, 0)); err != nil {
		t.Errorf("停止中の変更: %v", err)
	}
	if err := f.gdb.Exec("UPDATE users SET status = 'expired' WHERE id = ?", f.buyer2.ID).Error; err != nil {
		t.Fatal(err)
	}
	_, err = f.adminUsers.UpdateExpiresOn(ctx, f.admin, f.buyer2.ID, today.AddDate(1, 0, 0))
	wantCode(t, err, apperr.CodeInvalidStatusTransition)
}

// 退学による削除は、失効と同じ後処理を削除日から行う（AUTH-3, ADR 0033, ADR 0034）。
func TestAdminUserDelete(t *testing.T) {
	ctx := context.Background()
	f := newAdminFixture(t)
	inDeal := f.approved(t) // f.item を f.buyer と取引中
	requested := f.publish(t)
	req, err := f.deals.CreateRequest(ctx, f.buyer2, requested.ShareID, domain.PaymentMethodCard)
	if err != nil {
		t.Fatal(err)
	}
	published := f.publish(t)
	card := f.addCard(t, f.seller, "cur")
	f.addSession(t, f.seller, "seller-session")

	v, err := f.adminUsers.Delete(ctx, f.admin, f.seller.ID)
	if err != nil {
		t.Fatal(err)
	}
	if v.Status != domain.UserStatusDeleted {
		t.Errorf("status = %s", v.Status)
	}
	tr, err := f.txs.FindByID(ctx, inDeal.ID)
	if err != nil {
		t.Fatal(err)
	}
	if tr.Status != domain.TransactionStatusCancelled || tr.BuyerCancelAgreedAt != nil || tr.SellerCancelAgreedAt != nil {
		t.Errorf("取引中だった取引: %+v", tr)
	}
	wantKinds(t, f.paymentKinds(t, inDeal.ID), domain.PaymentKindAuthorize, domain.PaymentKindCapture, domain.PaymentKindRefund)
	if got := f.txStatus(t, req.ID); got != domain.TransactionStatusExpired {
		t.Errorf("購入希望 = %s, want expired", got)
	}
	for name, id := range map[string]string{"取引中だった商品": f.item.ID, "購入希望ありの商品": requested.ID, "公開中の商品": published.ID} {
		if got := f.itemStatusOf(t, id); got != domain.ItemStatusWithdrawn {
			t.Errorf("%s = %s, want withdrawn", name, got)
		}
	}
	// 学生証は削除日（日本時間の 2026-10-14）から3年
	got, err := f.cards.FindByID(ctx, card.ID)
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2029, 10, 14, 0, 0, 0, 0, time.UTC); got.DeleteAfter == nil || !got.DeleteAfter.Equal(want) {
		t.Errorf("削除予定日 = %v, want %v", got.DeleteAfter, want)
	}
	if f.sessionActive(t, "seller-session") {
		t.Error("セッションが残っている")
	}

	_, err = f.adminUsers.Delete(ctx, f.admin, f.seller.ID)
	wantCode(t, err, apperr.CodeInvalidStatusTransition)
	wantKinds(t, f.paymentKinds(t, inDeal.ID), domain.PaymentKindAuthorize, domain.PaymentKindCapture, domain.PaymentKindRefund)
}

// 同時に削除しても、後処理（返金）は1回だけ行う。
func TestAdminUserDelete_同時に削除(t *testing.T) {
	ctx := context.Background()
	f := newAdminFixture(t)
	inDeal := f.approved(t)

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := range errs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = f.adminUsers.Delete(ctx, f.admin, f.buyer.ID)
		}()
	}
	wg.Wait()
	ok := 0
	for _, err := range errs {
		if err == nil {
			ok++
		} else {
			wantCode(t, err, apperr.CodeInvalidStatusTransition)
		}
	}
	if ok != 1 {
		t.Errorf("成功した数 = %d, want 1（%v）", ok, errs)
	}
	wantKinds(t, f.paymentKinds(t, inDeal.ID), domain.PaymentKindAuthorize, domain.PaymentKindCapture, domain.PaymentKindRefund)
	// 購入者の削除では商品を公開中に戻す
	if got := f.itemStatus(t); got != domain.ItemStatusPublished {
		t.Errorf("商品 = %s, want published", got)
	}
}
