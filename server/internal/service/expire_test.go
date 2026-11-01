package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/domain"
)

// advanceTo は時計を at まで進める。
func (f *dealFixture) advanceTo(at time.Time) { f.clock.Advance(at.Sub(f.clock.Now())) }

func (f *dealFixture) expire(t *testing.T, want int) {
	t.Helper()
	if n, err := f.deals.ExpireRequests(context.Background()); err != nil || n != want {
		t.Fatalf("期限切れにした件数: n=%d err=%v want %d", n, err, want)
	}
}

// 承認期限（48時間）を過ぎた購入希望を期限切れにし、枠を解放する。承認待ちがなくなったら商品を公開中に戻す（DEAL-1, JOBS-1, ADR 0022）。
func TestDealExpireRequests(t *testing.T) {
	ctx := context.Background()
	f := newDealFixture(t)
	r1 := f.request(t, f.buyer)
	r2 := f.request(t, f.buyer2)

	f.advanceTo(r1.ApprovalDeadline.Add(-time.Second))
	f.expire(t, 0)
	if got := f.txStatus(t, r1.ID); got != domain.TransactionStatusRequested {
		t.Errorf("期限の1秒前: status = %s", got)
	}

	f.advanceTo(r1.ApprovalDeadline)
	f.expire(t, 1)
	if got := f.txStatus(t, r1.ID); got != domain.TransactionStatusExpired {
		t.Errorf("期限ちょうど: status = %s, want expired", got)
	}
	wantKinds(t, f.paymentKinds(t, r1.ID), domain.PaymentKindAuthorize, domain.PaymentKindRelease)
	if got := f.itemStatus(t); got != domain.ItemStatusRequested {
		t.Errorf("ほかに承認待ちがある: item = %s, want requested", got)
	}
	// 期限切れの購入希望は承認できない
	_, err := f.deals.Approve(ctx, f.seller, r1.ID)
	wantCode(t, err, apperr.CodeInvalidStatusTransition)

	f.advanceTo(r2.ApprovalDeadline)
	f.expire(t, 1)
	if got := f.itemStatus(t); got != domain.ItemStatusPublished {
		t.Errorf("承認待ちがなくなった: item = %s, want published", got)
	}
	// 何度呼んでも結果は同じ（枠の解放を二重に記録しない）
	f.expire(t, 0)
	wantKinds(t, f.paymentKinds(t, r2.ID), domain.PaymentKindAuthorize, domain.PaymentKindRelease)
}

// 承認待ちでない取引（取引中・却下済み）は期限を過ぎても触らない。
func TestDealExpireRequests_承認待ちでなければ触らない(t *testing.T) {
	ctx := context.Background()
	f := newDealFixture(t)
	deal := f.approved(t)
	// 却下済みは別の DB の商品で用意する
	f2 := newDealFixture(t)
	rejected := f2.request(t, f2.buyer)
	if _, err := f2.deals.Reject(ctx, f2.seller, rejected.ID); err != nil {
		t.Fatal(err)
	}

	for _, ff := range []*dealFixture{f, f2} {
		ff.advanceTo(t0.Add(domain.ApprovalTTL + time.Hour))
		ff.expire(t, 0)
	}
	if got := f.txStatus(t, deal.ID); got != domain.TransactionStatusInDeal {
		t.Errorf("取引中: status = %s", got)
	}
	if got := f.itemStatus(t); got != domain.ItemStatusInDeal {
		t.Errorf("取引中の商品: item = %s", got)
	}
	if got := f2.txStatus(t, rejected.ID); got != domain.TransactionStatusRejected {
		t.Errorf("却下済み: status = %s", got)
	}
	wantKinds(t, f2.paymentKinds(t, rejected.ID), domain.PaymentKindAuthorize, domain.PaymentKindRelease)
}

// 期限が切れた受け渡しトークンを消す（DEAL-3, JOBS-1）。
func TestDealDeleteExpiredHandoverTokens(t *testing.T) {
	ctx := context.Background()
	f := newDealFixture(t)
	tr := f.approved(t)
	f.issue(t, tr.ID, domain.HandoverTokenKindQR)
	f.issue(t, tr.ID, domain.HandoverTokenKindLink)

	f.clock.Advance(domain.HandoverQRTTL)
	if n, err := f.deals.DeleteExpiredHandoverTokens(ctx); err != nil || n != 1 {
		t.Fatalf("QRの期限の後: n=%d err=%v want 1", n, err)
	}
	f.clock.Advance(domain.HandoverIssuedTTL)
	if n, err := f.deals.DeleteExpiredHandoverTokens(ctx); err != nil || n != 1 {
		t.Fatalf("完了リンクの期限の後: n=%d err=%v want 1", n, err)
	}
}
