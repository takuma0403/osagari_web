package service_test

import (
	"context"
	"slices"
	"sync"
	"testing"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/service"
)

// dealOn は新しく出品した商品（¥1,500）に u が購入希望を出し、出品者が承認した取引を返す。
func (f *dealFixture) dealOn(t *testing.T, u *domain.User) *service.TransactionView {
	t.Helper()
	ctx := context.Background()
	v, err := f.deals.CreateRequest(ctx, u, f.publish(t).ShareID, domain.PaymentMethodCard)
	if err != nil {
		t.Fatal(err)
	}
	f.tick()
	if _, err := f.deals.Approve(ctx, f.seller, v.ID); err != nil {
		t.Fatal(err)
	}
	f.tick()
	return v
}

// completeDeal は出品者が QR を発行し、購入者 u が受け渡しを完了する。
func (f *dealFixture) completeDeal(t *testing.T, u *domain.User, id string) {
	t.Helper()
	tok := f.issue(t, id, domain.HandoverTokenKindQR)
	if _, err := f.deals.Complete(context.Background(), u, id, tokenCred(tok)); err != nil {
		t.Fatal(err)
	}
	f.tick()
}

func salesIDs(v *service.SalesView) []string {
	out := make([]string, len(v.Entries))
	for i, e := range v.Entries {
		out[i] = e.Transaction.ID
	}
	return out
}

// 売上の明細と集計：成立した取引だけを、記録済みの料率・手数料で最後に動いた順に返す（PAY, PAY-2, ADR 0029）。
func TestSalesGet(t *testing.T) {
	ctx := context.Background()
	f := newDealFixture(t)

	// 出品者の売上に出ないもの：承認待ちの購入希望・キャンセルした取引・ほかの人が出品者の取引
	f.request(t, f.buyer)
	cancelled := f.dealOn(t, f.buyer2)
	for _, u := range []*domain.User{f.buyer2, f.seller} {
		if _, err := f.deals.Cancel(ctx, u, cancelled.ID); err != nil {
			t.Fatal(err)
		}
	}
	f.tick()

	paidOut := f.dealOn(t, f.buyer)
	f.completeDeal(t, f.buyer, paidOut.ID)
	if _, err := f.sales.RequestPayout(ctx, f.seller); err != nil {
		t.Fatal(err)
	}
	f.tick()
	inDeal := f.dealOn(t, f.buyer)
	settled := f.dealOn(t, f.buyer2)
	f.completeDeal(t, f.buyer2, settled.ID)

	got, err := f.sales.Get(ctx, f.seller)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{settled.ID, inDeal.ID, paidOut.ID}; !slices.Equal(salesIDs(got), want) {
		t.Fatalf("並び = %v, want %v", salesIDs(got), want)
	}
	// ¥1,500 × 5% = 手数料 ¥75、売上 ¥1,425（ADR 0017 の暫定の料率）
	if got.SalesSummary != (domain.SalesSummary{Available: 1425, AwaitingHandover: 1425, PaidOut: 1425}) {
		t.Errorf("集計 = %+v", got.SalesSummary)
	}
	wantStates := []domain.SalesEntryState{domain.SalesEntryStateSettled, domain.SalesEntryStateAwaitingHandover, domain.SalesEntryStatePaidOut}
	for i, e := range got.Entries {
		tr := e.Transaction
		if e.State() != wantStates[i] {
			t.Errorf("%d: 状態 = %s, want %s", i, e.State(), wantStates[i])
		}
		if tr.Price != 1500 || *tr.FeeRateBP != 500 || *tr.Fee != 75 || *tr.SellerProceeds != 1425 {
			t.Errorf("%d: 金額 = %d, 料率 = %d, 手数料 = %d, 売上 = %d", i, tr.Price, *tr.FeeRateBP, *tr.Fee, *tr.SellerProceeds)
		}
		if e.Item == nil || e.Item.ID != tr.ItemID || len(e.Item.ImageURLs) != 1 {
			t.Errorf("%d: 商品 = %+v", i, e.Item)
		}
	}
	if e := got.Entries[2]; e.PaidOutAt == nil || e.Transaction.CompletedAt == nil {
		t.Errorf("出金済みの日時 = %v, 完了日時 = %v", e.PaidOutAt, e.Transaction.CompletedAt)
	}

	// 購入者や取引のない人には何も出ない
	for _, u := range []*domain.User{f.buyer, f.stranger} {
		v, err := f.sales.Get(ctx, u)
		if err != nil {
			t.Fatal(err)
		}
		if len(v.Entries) != 0 || v.SalesSummary != (domain.SalesSummary{}) {
			t.Errorf("出品者でない人の売上 = %+v", v)
		}
	}
}

// 出金申請：確定した売上だけを取引ごとに出金し、出金できるものがなければ 409（PAY, JOBS-2, ADR 0029）。
func TestSalesRequestPayout(t *testing.T) {
	ctx := context.Background()
	f := newDealFixture(t)

	// 受け渡し待ちだけでは出金できない
	inDeal := f.dealOn(t, f.buyer)
	if _, err := f.sales.RequestPayout(ctx, f.seller); apperr.CodeOf(err) != apperr.CodeInvalidStatusTransition {
		t.Fatalf("受け渡し待ちだけ: err = %v", err)
	}

	a := f.dealOn(t, f.buyer)
	f.completeDeal(t, f.buyer, a.ID)
	b := f.dealOn(t, f.buyer2)
	f.completeDeal(t, f.buyer2, b.ID)

	got, err := f.sales.RequestPayout(ctx, f.seller)
	if err != nil {
		t.Fatal(err)
	}
	if got.SalesSummary != (domain.SalesSummary{Available: 0, AwaitingHandover: 1425, PaidOut: 2850}) {
		t.Errorf("出金後の集計 = %+v", got.SalesSummary)
	}
	for _, id := range []string{a.ID, b.ID} {
		wantKinds(t, f.paymentKinds(t, id), domain.PaymentKindAuthorize, domain.PaymentKindCapture, domain.PaymentKindSettle, domain.PaymentKindPayout)
	}
	wantKinds(t, f.paymentKinds(t, inDeal.ID), domain.PaymentKindAuthorize, domain.PaymentKindCapture)

	// 出金済みの売上は出金し直さない
	if _, err := f.sales.RequestPayout(ctx, f.seller); apperr.CodeOf(err) != apperr.CodeInvalidStatusTransition {
		t.Errorf("出金済みの再申請: err = %v", err)
	}
	// ほかの人の申請では出品者の売上は出金されない
	if _, err := f.sales.RequestPayout(ctx, f.buyer); apperr.CodeOf(err) != apperr.CodeInvalidStatusTransition {
		t.Errorf("購入者の申請: err = %v", err)
	}
}

// 同時に2回出金を申請しても、出金は取引ごとに1回だけ記録される（DB, ADR 0029）。
func TestSalesRequestPayout_同時(t *testing.T) {
	ctx := context.Background()
	f := newDealFixture(t)
	tr := f.dealOn(t, f.buyer)
	f.completeDeal(t, f.buyer, tr.ID)

	const n = 2
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() { _, errs[i] = f.sales.RequestPayout(ctx, f.seller) })
	}
	wg.Wait()

	ok := 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case apperr.CodeOf(err) != apperr.CodeInvalidStatusTransition:
			t.Errorf("想定外のエラー: %v", err)
		}
	}
	if ok != 1 {
		t.Errorf("成功した申請 = %d, want 1（errs = %v）", ok, errs)
	}
	wantKinds(t, f.paymentKinds(t, tr.ID), domain.PaymentKindAuthorize, domain.PaymentKindCapture, domain.PaymentKindSettle, domain.PaymentKindPayout)
}

// マイページの出品の一覧：本人のすべての出品を最後に更新した順に、承認待ちの件数つきで返す（AUTH-3, ADR 0029）。
func TestItemListOwn(t *testing.T) {
	ctx := context.Background()
	f := newDealFixture(t)
	f.request(t, f.buyer)
	f.request(t, f.buyer2)
	draft, err := f.svc.Create(ctx, f.seller, domain.ItemContent{Title: "下書き"}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	f.tick()
	deal := f.dealOn(t, f.buyer)

	got, err := f.svc.ListOwn(ctx, f.seller)
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, len(got))
	for i, v := range got {
		ids[i] = v.Item.ID
	}
	dealTr, err := f.txs.FindByID(ctx, deal.ID)
	if err != nil {
		t.Fatal(err)
	}
	// 最初の商品は購入希望で更新された後に下書き・取引中の商品が作られたので、いちばん後ろになる
	if want := []string{dealTr.ItemID, draft.ID, f.item.ID}; !slices.Equal(ids, want) {
		t.Fatalf("並び = %v, want %v", ids, want)
	}
	wantCounts := []int{0, 0, 2}
	for i, v := range got {
		if v.RequestCount != wantCounts[i] {
			t.Errorf("%d: 承認待ちの件数 = %d, want %d", i, v.RequestCount, wantCounts[i])
		}
	}
	if s := got[2].Item.Status; s != domain.ItemStatusRequested {
		t.Errorf("購入希望ありの商品のステータス = %s", s)
	}
	if len(got[0].Item.ImageURLs) != 1 || len(got[1].Item.ImageURLs) != 0 {
		t.Errorf("サムネイル = %d枚, %d枚", len(got[0].Item.ImageURLs), len(got[1].Item.ImageURLs))
	}

	// 出品のない人は空
	if v, err := f.svc.ListOwn(ctx, f.buyer); err != nil || len(v) != 0 {
		t.Errorf("出品のない人: got = %v, err = %v", v, err)
	}
}
