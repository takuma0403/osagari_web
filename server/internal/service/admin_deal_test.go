package service_test

import (
	"context"
	"sync"
	"testing"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/service"
)

// 出品・取引の一覧は、すべてを新しい順に、出品者・購入者の学籍番号つきで返す（ADMIN）。
func TestAdminDealLists(t *testing.T) {
	ctx := context.Background()
	f := newAdminFixture(t)
	tr := f.request(t, f.buyer)
	draft, err := f.svc.Create(ctx, f.seller, domain.ItemContent{Title: "下書き"}, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	items, err := f.adminDeals.ListItems(ctx, f.admin, service.AdminPageQuery{Limit: ptr(1)})
	if err != nil {
		t.Fatal(err)
	}
	if len(items.Entries) != 1 || items.Entries[0].Item.ID != draft.ID || items.Entries[0].SellerStudentNumber != f.seller.StudentNumber || items.NextCursor == nil {
		t.Fatalf("出品の1ページ目: %+v", items)
	}
	items, err = f.adminDeals.ListItems(ctx, f.admin, service.AdminPageQuery{Cursor: items.NextCursor})
	if err != nil || len(items.Entries) != 1 || items.Entries[0].Item.ID != f.item.ID || len(items.Entries[0].Item.ImageURLs) != 1 {
		t.Fatalf("出品の続き: %+v err=%v", items, err)
	}

	// 管理者には承認前でも学籍番号を見せる
	trs, err := f.adminDeals.ListTransactions(ctx, f.admin, service.AdminPageQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if len(trs.Entries) != 1 {
		t.Fatalf("取引の一覧: %+v", trs)
	}
	if e := trs.Entries[0]; e.ID != tr.ID || e.BuyerStudentNumber != f.buyer.StudentNumber || e.SellerStudentNumber != f.seller.StudentNumber || e.Item.ID != f.item.ID {
		t.Errorf("取引: %+v", e)
	}
}

// 出品の停止：購入希望ありなら購入希望を却下して枠を解放し、取引中なら取引を続ける（ITEM-1, ADR 0034）。
func TestAdminSuspendItem(t *testing.T) {
	ctx := context.Background()

	t.Run("公開中", func(t *testing.T) {
		f := newAdminFixture(t)
		v, err := f.adminDeals.SuspendItem(ctx, f.admin, f.item.ID)
		if err != nil || v.Item.Status != domain.ItemStatusSuspended {
			t.Fatalf("%+v err=%v", v, err)
		}
		_, err = f.adminDeals.SuspendItem(ctx, f.admin, f.item.ID)
		wantCode(t, err, apperr.CodeInvalidStatusTransition)
	})

	t.Run("購入希望あり", func(t *testing.T) {
		f := newAdminFixture(t)
		req := f.request(t, f.buyer)
		if _, err := f.adminDeals.SuspendItem(ctx, f.admin, f.item.ID); err != nil {
			t.Fatal(err)
		}
		if got := f.txStatus(t, req.ID); got != domain.TransactionStatusRejected {
			t.Errorf("購入希望 = %s, want rejected", got)
		}
		wantKinds(t, f.paymentKinds(t, req.ID), domain.PaymentKindAuthorize, domain.PaymentKindRelease)
	})

	t.Run("取引中は取引を続け、完了できる", func(t *testing.T) {
		f := newAdminFixture(t)
		tr := f.approved(t)
		if _, err := f.adminDeals.SuspendItem(ctx, f.admin, f.item.ID); err != nil {
			t.Fatal(err)
		}
		if got := f.txStatus(t, tr.ID); got != domain.TransactionStatusInDeal {
			t.Errorf("取引 = %s, want in_deal", got)
		}
		f.completeDeal(t, f.buyer, tr.ID)
		if got := f.itemStatus(t); got != domain.ItemStatusSuspended {
			t.Errorf("商品 = %s, want suspended", got)
		}
	})

	t.Run("取り下げ済み・存在しない", func(t *testing.T) {
		f := newAdminFixture(t)
		f.setStatus(t, f.item.ID, domain.ItemStatusWithdrawn)
		_, err := f.adminDeals.SuspendItem(ctx, f.admin, f.item.ID)
		wantCode(t, err, apperr.CodeInvalidStatusTransition)
		_, err = f.adminDeals.SuspendItem(ctx, f.admin, "missing")
		wantCode(t, err, apperr.CodeNotFound)
	})
}

// 管理者のキャンセル：取引中だけ。返金して商品を公開中に戻し、同意日時は入れない（キャンセル回数に数えない）（DEAL-4, ADR 0034）。
func TestAdminCancelTransaction(t *testing.T) {
	ctx := context.Background()

	t.Run("取引中", func(t *testing.T) {
		f := newAdminFixture(t)
		tr := f.approved(t)
		if _, err := f.deals.Cancel(ctx, f.buyer, tr.ID); err != nil { // 購入者だけが申請していた
			t.Fatal(err)
		}
		v, err := f.adminDeals.CancelTransaction(ctx, f.admin, tr.ID)
		if err != nil {
			t.Fatal(err)
		}
		if v.Status != domain.TransactionStatusCancelled || v.SellerCancelAgreedAt != nil {
			t.Errorf("取引: %+v", v.Transaction)
		}
		if got := f.itemStatus(t); got != domain.ItemStatusPublished {
			t.Errorf("商品 = %s, want published", got)
		}
		wantKinds(t, f.paymentKinds(t, tr.ID), domain.PaymentKindAuthorize, domain.PaymentKindCapture, domain.PaymentKindRefund)
		u, err := f.adminUsers.Get(ctx, f.admin, f.buyer.ID)
		if err != nil || u.CancelCount != 0 {
			t.Errorf("キャンセル回数: %+v err=%v", u, err)
		}
		_, err = f.adminDeals.CancelTransaction(ctx, f.admin, tr.ID)
		wantCode(t, err, apperr.CodeInvalidStatusTransition)
	})

	t.Run("停止中の商品はそのまま", func(t *testing.T) {
		f := newAdminFixture(t)
		tr := f.approved(t)
		if _, err := f.adminDeals.SuspendItem(ctx, f.admin, f.item.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := f.adminDeals.CancelTransaction(ctx, f.admin, tr.ID); err != nil {
			t.Fatal(err)
		}
		if got := f.itemStatus(t); got != domain.ItemStatusSuspended {
			t.Errorf("商品 = %s, want suspended", got)
		}
	})

	t.Run("完了後・購入希望・存在しない", func(t *testing.T) {
		f := newAdminFixture(t)
		done := f.dealOn(t, f.buyer2)
		f.completeDeal(t, f.buyer2, done.ID)
		_, err := f.adminDeals.CancelTransaction(ctx, f.admin, done.ID)
		wantCode(t, err, apperr.CodeInvalidStatusTransition)
		req := f.request(t, f.buyer)
		_, err = f.adminDeals.CancelTransaction(ctx, f.admin, req.ID)
		wantCode(t, err, apperr.CodeInvalidStatusTransition)
		_, err = f.adminDeals.CancelTransaction(ctx, f.admin, "missing")
		wantCode(t, err, apperr.CodeNotFound)
	})

	t.Run("同時にキャンセルしても返金は1回", func(t *testing.T) {
		f := newAdminFixture(t)
		tr := f.approved(t)
		var wg sync.WaitGroup
		errs := make([]error, 2)
		for i := range errs {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, errs[i] = f.adminDeals.CancelTransaction(ctx, f.admin, tr.ID)
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
		wantKinds(t, f.paymentKinds(t, tr.ID), domain.PaymentKindAuthorize, domain.PaymentKindCapture, domain.PaymentKindRefund)
	})
}

// 手数料率の変更は、変更後に成立する取引にだけ使われる（PAY-2, ADR 0003, ADR 0034）。
func TestAdminUpdateFeeRate(t *testing.T) {
	ctx := context.Background()
	f := newAdminFixture(t)

	v, err := f.adminSetting.AdminFeeRate(ctx, f.admin)
	if err != nil || v.BasisPoints != 500 || v.UpdatedByStudentNumber != nil {
		t.Fatalf("初期値: %+v err=%v", v, err)
	}
	before := f.approved(t) // 5% で成立

	f.tick()
	v, err = f.adminSetting.UpdateFeeRate(ctx, f.admin, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if v.BasisPoints != 1000 || v.UpdatedByStudentNumber == nil || *v.UpdatedByStudentNumber != adminNumber || !v.UpdatedAt.Equal(f.clock.Now()) {
		t.Errorf("変更後: %+v", v)
	}
	after := f.dealOn(t, f.buyer2) // 10% で成立

	for _, c := range []struct {
		name string
		id   string
		bp   int
		fee  int64
	}{{"変更前に成立", before.ID, 500, 75}, {"変更後に成立", after.ID, 1000, 150}} {
		tr, err := f.txs.FindByID(ctx, c.id)
		if err != nil {
			t.Fatal(err)
		}
		if tr.FeeRateBP == nil || *tr.FeeRateBP != c.bp || *tr.Fee != c.fee {
			t.Errorf("%s: 料率=%v 手数料=%v, want %d / %d", c.name, tr.FeeRateBP, tr.Fee, c.bp, c.fee)
		}
	}

	for _, bp := range []int{-1, 10001} {
		_, err := f.adminSetting.UpdateFeeRate(ctx, f.admin, bp)
		wantCode(t, err, apperr.CodeBadRequest)
	}
	for _, bp := range []int{0, 10000} {
		if _, err := f.adminSetting.UpdateFeeRate(ctx, f.admin, bp); err != nil {
			t.Errorf("境界値 %d: %v", bp, err)
		}
	}
}
