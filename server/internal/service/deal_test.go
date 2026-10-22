package service_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/service"
)

// dealFixture は公開中の商品（¥1,500）と、購入者2人・取引に関係しない利用者をそろえる。
type dealFixture struct {
	*itemFixture
	item                    *service.ItemView
	buyer, buyer2, stranger *domain.User
}

func newDealFixture(t *testing.T) *dealFixture {
	t.Helper()
	f := &dealFixture{itemFixture: newItemFixture(t)}
	f.item = f.publish(t)
	// 購入者は学生証の承認がなくても購入希望を出せる
	f.buyer = f.createUser(t, "26141023@edu.cc.saga-u.ac.jp", "26141023", domain.SellerStatusNone)
	f.buyer2 = f.createUser(t, "25141024@edu.cc.saga-u.ac.jp", "25141024", domain.SellerStatusNone)
	f.stranger = f.createUser(t, "26141025@edu.cc.saga-u.ac.jp", "26141025", domain.SellerStatusNone)
	return f
}

// request は u が商品に購入希望を出し、時計を1分進める。
func (f *dealFixture) request(t *testing.T, u *domain.User) *service.TransactionView {
	t.Helper()
	v, err := f.deals.CreateRequest(context.Background(), u, f.item.ShareID, domain.PaymentMethodCard)
	if err != nil {
		t.Fatal(err)
	}
	f.tick()
	return v
}

func (f *dealFixture) itemStatus(t *testing.T) domain.ItemStatus {
	t.Helper()
	it, err := f.items.FindByID(context.Background(), f.item.ID)
	if err != nil {
		t.Fatal(err)
	}
	return it.Status
}

func (f *dealFixture) txStatus(t *testing.T, id string) domain.TransactionStatus {
	t.Helper()
	tr, err := f.txs.FindByID(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return tr.Status
}

// paymentKinds は取引の決済の記録の種別を古い順に返す（JOBS-2）。
func (f *dealFixture) paymentKinds(t *testing.T, id string) []domain.PaymentKind {
	t.Helper()
	ps, err := f.payments.ListByTransaction(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	kinds := make([]domain.PaymentKind, len(ps))
	for i, p := range ps {
		kinds[i] = p.Kind
	}
	return kinds
}

func wantKinds(t *testing.T, got []domain.PaymentKind, want ...domain.PaymentKind) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("決済の記録 = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("決済の記録 = %v, want %v", got, want)
		}
	}
}

// wantUndisclosed は承認前の取引に学籍番号・手数料が入っていないことを確かめる（DEAL-1）。
func wantUndisclosed(t *testing.T, v *service.TransactionView) {
	t.Helper()
	if v.CounterpartStudentNumber != nil || v.IdentityDisclosedAt != nil || v.FeeRateBP != nil || v.Fee != nil || v.SellerProceeds != nil {
		t.Errorf("承認前なのに開示されている: %+v", v.Transaction)
	}
}

func TestDealCreateRequest(t *testing.T) {
	ctx := context.Background()

	t.Run("購入希望で枠を確保し、商品は購入希望ありになる", func(t *testing.T) {
		f := newDealFixture(t)
		v, err := f.deals.CreateRequest(ctx, f.buyer, f.item.ShareID, domain.PaymentMethodPayPay)
		if err != nil {
			t.Fatal(err)
		}
		if v.Role != domain.TransactionRoleBuyer || v.Status != domain.TransactionStatusRequested || v.Price != 1500 ||
			v.PaymentMethod != domain.PaymentMethodPayPay || v.SellerID != f.seller.ID || v.Item.ID != f.item.ID ||
			!v.ApprovalDeadline.Equal(t0.Add(48*time.Hour)) {
			t.Errorf("got = %+v", v.Transaction)
		}
		wantUndisclosed(t, v)
		if got := f.itemStatus(t); got != domain.ItemStatusRequested {
			t.Errorf("商品のステータス = %s", got)
		}
		wantKinds(t, f.paymentKinds(t, v.ID), domain.PaymentKindAuthorize)

		// 購入希望ありの間もほかの人は購入希望を出せる
		f.request(t, f.buyer2)
		if got := f.itemStatus(t); got != domain.ItemStatusRequested {
			t.Errorf("2件目の後の商品のステータス = %s", got)
		}
	})

	t.Run("出せない場合", func(t *testing.T) {
		f := newDealFixture(t)
		f.request(t, f.buyer)

		_, err := f.deals.CreateRequest(ctx, f.buyer, f.item.ShareID, domain.PaymentMethodCard)
		wantCode(t, err, apperr.CodeInvalidStatusTransition)
		_, err = f.deals.CreateRequest(ctx, f.seller, f.item.ShareID, domain.PaymentMethodCard)
		wantCode(t, err, apperr.CodeForbidden)
		_, err = f.deals.CreateRequest(ctx, f.buyer2, f.item.ShareID, "cash")
		wantCode(t, err, apperr.CodeBadRequest)
		_, err = f.deals.CreateRequest(ctx, f.buyer2, "no-such-share", domain.PaymentMethodCard)
		wantCode(t, err, apperr.CodeNotFound)
	})

	t.Run("商品のステータスごと", func(t *testing.T) {
		tests := []struct {
			status domain.ItemStatus
			code   apperr.Code
		}{
			{domain.ItemStatusPublished, ""},
			{domain.ItemStatusRequested, ""},
			{domain.ItemStatusInDeal, apperr.CodeInvalidStatusTransition},
			{domain.ItemStatusCompleted, apperr.CodeInvalidStatusTransition},
			{domain.ItemStatusDraft, apperr.CodeNotFound},
			{domain.ItemStatusWithdrawn, apperr.CodeNotFound},
			{domain.ItemStatusSuspended, apperr.CodeNotFound},
		}
		for _, tt := range tests {
			t.Run(string(tt.status), func(t *testing.T) {
				f := newDealFixture(t)
				f.setStatus(t, f.item.ID, tt.status)
				_, err := f.deals.CreateRequest(ctx, f.buyer, f.item.ShareID, domain.PaymentMethodCard)
				wantCode(t, err, tt.code)
			})
		}
	})
}

func TestDealListRequests(t *testing.T) {
	ctx := context.Background()
	f := newDealFixture(t)
	r1 := f.request(t, f.buyer)
	r2 := f.request(t, f.buyer2)

	got, err := f.deals.ListRequests(ctx, f.seller, f.item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != r1.ID || got[1].ID != r2.ID {
		t.Fatalf("一覧 = %+v", got)
	}
	if b := got[1].Buyer; b.AdmissionYear != 2026 || b.CompletedDealCount != 0 {
		t.Errorf("購入者の情報 = %+v", b)
	}

	// 却下した購入希望は一覧から消える
	if _, err := f.deals.Reject(ctx, f.seller, r1.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := f.deals.ListRequests(ctx, f.seller, f.item.ID); len(got) != 1 || got[0].ID != r2.ID {
		t.Errorf("却下の後の一覧 = %+v", got)
	}

	// 出品者でなければ商品の存在も区別しない
	for _, u := range []*domain.User{f.buyer, f.stranger} {
		_, err := f.deals.ListRequests(ctx, u, f.item.ID)
		wantCode(t, err, apperr.CodeNotFound)
	}
}

func TestDealApprove(t *testing.T) {
	ctx := context.Background()

	t.Run("承認で代金を預かり、学籍番号を開示し、ほかの購入希望を却下する", func(t *testing.T) {
		f := newDealFixture(t)
		r1 := f.request(t, f.buyer)
		r2 := f.request(t, f.buyer2)

		// 承認前はどちらからも学籍番号が見えない
		for _, u := range []*domain.User{f.seller, f.buyer} {
			v, err := f.deals.Get(ctx, u, r1.ID)
			if err != nil {
				t.Fatal(err)
			}
			wantUndisclosed(t, v)
		}

		v, err := f.deals.Approve(ctx, f.seller, r1.ID)
		if err != nil {
			t.Fatal(err)
		}
		// 手数料は暫定の 5%（ADR 0017）
		if v.Role != domain.TransactionRoleSeller || v.Status != domain.TransactionStatusInDeal ||
			*v.FeeRateBP != 500 || *v.Fee != 75 || *v.SellerProceeds != 1425 ||
			!v.IdentityDisclosedAt.Equal(f.clock.Now()) || v.CounterpartStudentNumber == nil || *v.CounterpartStudentNumber != f.buyer.StudentNumber {
			t.Errorf("出品者から見た取引 = %+v, 学籍番号 = %v", v.Transaction, v.CounterpartStudentNumber)
		}
		b, err := f.deals.Get(ctx, f.buyer, r1.ID)
		if err != nil {
			t.Fatal(err)
		}
		if b.CounterpartStudentNumber == nil || *b.CounterpartStudentNumber != f.seller.StudentNumber || b.Fee != nil {
			t.Errorf("購入者から見た取引 = %+v, 学籍番号 = %v", b.Transaction, b.CounterpartStudentNumber)
		}

		if got := f.itemStatus(t); got != domain.ItemStatusInDeal {
			t.Errorf("商品のステータス = %s", got)
		}
		if got := f.txStatus(t, r2.ID); got != domain.TransactionStatusRejected {
			t.Errorf("ほかの購入希望 = %s", got)
		}
		wantKinds(t, f.paymentKinds(t, r1.ID), domain.PaymentKindAuthorize, domain.PaymentKindCapture)
		wantKinds(t, f.paymentKinds(t, r2.ID), domain.PaymentKindAuthorize, domain.PaymentKindRelease)

		// 却下された購入者には学籍番号を開示しない
		rv, err := f.deals.Get(ctx, f.buyer2, r2.ID)
		if err != nil {
			t.Fatal(err)
		}
		wantUndisclosed(t, rv)
	})

	t.Run("承認できない場合", func(t *testing.T) {
		tests := []struct {
			name string
			// prepare は承認の前に状態を作り、承認する人を返す
			prepare func(t *testing.T, f *dealFixture, tr *service.TransactionView) *domain.User
			code    apperr.Code
		}{
			{"購入者", func(_ *testing.T, f *dealFixture, _ *service.TransactionView) *domain.User { return f.buyer }, apperr.CodeNotFound},
			{"取引に関係しない人", func(_ *testing.T, f *dealFixture, _ *service.TransactionView) *domain.User { return f.stranger }, apperr.CodeNotFound},
			{"承認期限を過ぎた", func(_ *testing.T, f *dealFixture, _ *service.TransactionView) *domain.User {
				f.clock.Advance(48 * time.Hour)
				return f.seller
			}, apperr.CodeInvalidStatusTransition},
			{"承認済み", func(t *testing.T, f *dealFixture, tr *service.TransactionView) *domain.User {
				if _, err := f.deals.Approve(context.Background(), f.seller, tr.ID); err != nil {
					t.Fatal(err)
				}
				return f.seller
			}, apperr.CodeInvalidStatusTransition},
			{"却下済み", func(t *testing.T, f *dealFixture, tr *service.TransactionView) *domain.User {
				if _, err := f.deals.Reject(context.Background(), f.seller, tr.ID); err != nil {
					t.Fatal(err)
				}
				return f.seller
			}, apperr.CodeInvalidStatusTransition},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				f := newDealFixture(t)
				tr := f.request(t, f.buyer)
				u := tt.prepare(t, f, tr)
				_, err := f.deals.Approve(ctx, u, tr.ID)
				wantCode(t, err, tt.code)
			})
		}
	})

	t.Run("同じ商品の購入希望を同時に承認しても成立は1件だけ", func(t *testing.T) {
		f := newDealFixture(t)
		reqs := []*service.TransactionView{f.request(t, f.buyer), f.request(t, f.buyer2)}
		const n = 8
		var wg sync.WaitGroup
		errs := make([]error, n)
		for i := range n {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, errs[i] = f.deals.Approve(ctx, f.seller, reqs[i%len(reqs)].ID)
			}()
		}
		wg.Wait()
		ok := 0
		for _, err := range errs {
			switch apperr.CodeOf(err) {
			case "":
				ok++
			case apperr.CodeInvalidStatusTransition:
			default:
				t.Errorf("err = %v", err)
			}
		}
		if ok != 1 {
			t.Errorf("成功 = %d, want 1", ok)
		}
		inDeal := 0
		for _, r := range reqs {
			if f.txStatus(t, r.ID) == domain.TransactionStatusInDeal {
				inDeal++
			}
		}
		if inDeal != 1 {
			t.Errorf("取引中の取引 = %d, want 1", inDeal)
		}
	})
}

func TestDealReject(t *testing.T) {
	ctx := context.Background()
	f := newDealFixture(t)
	r1 := f.request(t, f.buyer)
	r2 := f.request(t, f.buyer2)

	for _, u := range []*domain.User{f.buyer, f.stranger} {
		_, err := f.deals.Reject(ctx, u, r1.ID)
		wantCode(t, err, apperr.CodeNotFound)
	}

	v, err := f.deals.Reject(ctx, f.seller, r1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if v.Status != domain.TransactionStatusRejected {
		t.Errorf("却下: got = %+v", v.Transaction)
	}
	wantUndisclosed(t, v)
	wantKinds(t, f.paymentKinds(t, r1.ID), domain.PaymentKindAuthorize, domain.PaymentKindRelease)
	// ほかに購入希望が残っていれば購入希望ありのまま
	if got := f.itemStatus(t); got != domain.ItemStatusRequested {
		t.Errorf("1件目の却下の後の商品 = %s", got)
	}
	_, err = f.deals.Reject(ctx, f.seller, r1.ID)
	wantCode(t, err, apperr.CodeInvalidStatusTransition)

	// 購入希望がなくなったら公開中に戻す。公開日時（新着順）は変えない（ADR 0022）
	if _, err := f.deals.Reject(ctx, f.seller, r2.ID); err != nil {
		t.Fatal(err)
	}
	it, err := f.items.FindByID(ctx, f.item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if it.Status != domain.ItemStatusPublished || !it.PublishedAt.Equal(*f.item.PublishedAt) {
		t.Errorf("すべて却下の後の商品: status = %s, publishedAt = %v", it.Status, it.PublishedAt)
	}

	// 却下した購入者はもう一度購入希望を出せる
	f.request(t, f.buyer)
}

func TestDealGet(t *testing.T) {
	ctx := context.Background()
	f := newDealFixture(t)
	tr := f.request(t, f.buyer)

	_, err := f.deals.Get(ctx, f.stranger, tr.ID)
	wantCode(t, err, apperr.CodeNotFound)
	_, err = f.deals.Get(ctx, f.buyer, "no-such-id")
	wantCode(t, err, apperr.CodeNotFound)
	v, err := f.deals.Get(ctx, f.seller, tr.ID)
	if err != nil {
		t.Fatal(err)
	}
	if v.Role != domain.TransactionRoleSeller {
		t.Errorf("role = %s", v.Role)
	}
}

func TestDealAcknowledge(t *testing.T) {
	ctx := context.Background()
	f := newDealFixture(t)
	tr := f.request(t, f.buyer)

	_, err := f.deals.Acknowledge(ctx, f.seller, tr.ID)
	wantCode(t, err, apperr.CodeInvalidStatusTransition)
	if _, err := f.deals.Approve(ctx, f.seller, tr.ID); err != nil {
		t.Fatal(err)
	}
	_, err = f.deals.Acknowledge(ctx, f.buyer, tr.ID)
	wantCode(t, err, apperr.CodeNotFound)

	at := f.clock.Now()
	v, err := f.deals.Acknowledge(ctx, f.seller, tr.ID)
	if err != nil {
		t.Fatal(err)
	}
	if v.SellerAcknowledgedAt == nil || !v.SellerAcknowledgedAt.Equal(at) {
		t.Errorf("確認の日時 = %v", v.SellerAcknowledgedAt)
	}
	// もう一度押しても最初の日時のまま
	f.tick()
	v, err = f.deals.Acknowledge(ctx, f.seller, tr.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !v.SellerAcknowledgedAt.Equal(at) {
		t.Errorf("2回目の確認の日時 = %v, want %v", v.SellerAcknowledgedAt, at)
	}
}

// 購入希望ありの商品を取り下げると、購入希望をすべて却下して枠を解放する（ADR 0022）。
func TestItemWithdrawRejectsRequests(t *testing.T) {
	ctx := context.Background()
	f := newDealFixture(t)
	r1 := f.request(t, f.buyer)
	r2 := f.request(t, f.buyer2)

	if _, err := f.svc.Withdraw(ctx, f.seller, f.item.ID); err != nil {
		t.Fatal(err)
	}
	for _, r := range []*service.TransactionView{r1, r2} {
		if got := f.txStatus(t, r.ID); got != domain.TransactionStatusRejected {
			t.Errorf("購入希望 = %s", got)
		}
		wantKinds(t, f.paymentKinds(t, r.ID), domain.PaymentKindAuthorize, domain.PaymentKindRelease)
	}
	_, err := f.deals.Approve(ctx, f.seller, r1.ID)
	wantCode(t, err, apperr.CodeInvalidStatusTransition)
}
