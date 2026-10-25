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

// approved は購入者の購入希望を出品者が承認した取引中の取引を返し、時計を1分進める。
func (f *dealFixture) approved(t *testing.T) *service.TransactionView {
	t.Helper()
	tr := f.request(t, f.buyer)
	if _, err := f.deals.Approve(context.Background(), f.seller, tr.ID); err != nil {
		t.Fatal(err)
	}
	// 決済の記録の並び（作った日時の順）を、承認と完了で入れ替わらないようにする
	f.tick()
	return tr
}

// issue は出品者が受け渡しトークンを発行する。
func (f *dealFixture) issue(t *testing.T, id string, kind domain.HandoverTokenKind) *service.IssuedHandoverToken {
	t.Helper()
	tok, err := f.deals.IssueHandoverToken(context.Background(), f.seller, id, kind)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func tokenCred(tok *service.IssuedHandoverToken) service.HandoverCredential {
	return service.HandoverCredential{Kind: tok.Kind, Secret: *tok.Token}
}

func codeCred(tok *service.IssuedHandoverToken) service.HandoverCredential {
	return service.HandoverCredential{Kind: domain.HandoverTokenKindCode, Secret: *tok.Code}
}

func TestDealIssueHandoverToken(t *testing.T) {
	ctx := context.Background()

	t.Run("種類ごとの中身と有効期限", func(t *testing.T) {
		f := newDealFixture(t)
		tr := f.approved(t)
		tests := []struct {
			kind              domain.HandoverTokenKind
			hasToken, hasCode bool
			ttl               time.Duration
		}{
			{domain.HandoverTokenKindQR, true, true, 5 * time.Minute},
			{domain.HandoverTokenKindLink, true, false, 24 * time.Hour},
			{domain.HandoverTokenKindCode, false, true, 24 * time.Hour},
		}
		for _, tt := range tests {
			tok := f.issue(t, tr.ID, tt.kind)
			if (tok.Token != nil) != tt.hasToken || (tok.Code != nil) != tt.hasCode || !tok.ExpiresAt.Equal(f.clock.Now().Add(tt.ttl)) {
				t.Errorf("%s: %+v", tt.kind, tok)
			}
			if tok.Code != nil && len(*tok.Code) != 6 {
				t.Errorf("%s: コード = %q", tt.kind, *tok.Code)
			}
		}
	})

	t.Run("発行できない場合", func(t *testing.T) {
		f := newDealFixture(t)
		requested := f.request(t, f.buyer2)
		tr := f.approved(t)
		tests := []struct {
			name string
			u    *domain.User
			id   string
			kind domain.HandoverTokenKind
			code apperr.Code
		}{
			{"購入者", f.buyer, tr.ID, domain.HandoverTokenKindQR, apperr.CodeNotFound},
			{"取引に関係しない人", f.stranger, tr.ID, domain.HandoverTokenKindQR, apperr.CodeNotFound},
			// 承認で却下された購入希望（取引中でない）
			{"取引中でない", f.seller, requested.ID, domain.HandoverTokenKindQR, apperr.CodeInvalidStatusTransition},
			{"種類が不正", f.seller, tr.ID, "auto", apperr.CodeBadRequest},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := f.deals.IssueHandoverToken(ctx, tt.u, tt.id, tt.kind)
				wantCode(t, err, tt.code)
			})
		}
	})

	t.Run("同じ種類を発行し直すと前のトークンは使えない", func(t *testing.T) {
		f := newDealFixture(t)
		tr := f.approved(t)
		old := f.issue(t, tr.ID, domain.HandoverTokenKindQR)
		link := f.issue(t, tr.ID, domain.HandoverTokenKindLink)
		cur := f.issue(t, tr.ID, domain.HandoverTokenKindQR)
		wantCode(t, f.deals.VerifyHandover(ctx, f.buyer, tr.ID, tokenCred(old)), service.CodeInvalidHandoverToken)
		if *old.Code != *cur.Code {
			wantCode(t, f.deals.VerifyHandover(ctx, f.buyer, tr.ID, codeCred(old)), service.CodeInvalidHandoverToken)
		}
		for _, c := range []service.HandoverCredential{tokenCred(cur), codeCred(cur), tokenCred(link)} {
			if err := f.deals.VerifyHandover(ctx, f.buyer, tr.ID, c); err != nil {
				t.Errorf("%s: %v", c.Kind, err)
			}
		}
	})
}

func TestDealVerifyHandover(t *testing.T) {
	ctx := context.Background()
	f := newDealFixture(t)
	tr := f.approved(t)
	qr := f.issue(t, tr.ID, domain.HandoverTokenKindQR)
	link := f.issue(t, tr.ID, domain.HandoverTokenKindLink)
	code := f.issue(t, tr.ID, domain.HandoverTokenKindCode)

	for _, c := range []service.HandoverCredential{tokenCred(qr), codeCred(qr), tokenCred(link), codeCred(code)} {
		// 照合では使用済みにしないので、何度でも通る
		for range 2 {
			if err := f.deals.VerifyHandover(ctx, f.buyer, tr.ID, c); err != nil {
				t.Errorf("%s: %v", c.Kind, err)
			}
		}
	}

	tests := []struct {
		name string
		u    *domain.User
		cred service.HandoverCredential
		code apperr.Code
	}{
		{"出品者", f.seller, tokenCred(qr), apperr.CodeNotFound},
		{"取引に関係しない人", f.stranger, tokenCred(qr), apperr.CodeNotFound},
		{"違うトークン", f.buyer, service.HandoverCredential{Kind: domain.HandoverTokenKindQR, Secret: "x"}, service.CodeInvalidHandoverToken},
		{"空のトークン", f.buyer, service.HandoverCredential{Kind: domain.HandoverTokenKindLink}, service.CodeInvalidHandoverToken},
		{"リンクのトークンを QR として", f.buyer, service.HandoverCredential{Kind: domain.HandoverTokenKindQR, Secret: *link.Token}, service.CodeInvalidHandoverToken},
		{"違うコード", f.buyer, service.HandoverCredential{Kind: domain.HandoverTokenKindCode, Secret: wrongCode2(*qr.Code, *code.Code)}, service.CodeInvalidHandoverToken},
		{"6桁でないコード", f.buyer, service.HandoverCredential{Kind: domain.HandoverTokenKindCode, Secret: "12345"}, service.CodeInvalidHandoverToken},
		{"種類が不正", f.buyer, service.HandoverCredential{Kind: "auto", Secret: *qr.Token}, apperr.CodeBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantCode(t, f.deals.VerifyHandover(ctx, tt.u, tr.ID, tt.cred), tt.code)
		})
	}

	t.Run("ほかの取引のトークン", func(t *testing.T) {
		g := newDealFixture(t)
		other := g.approved(t)
		tok := g.issue(t, other.ID, domain.HandoverTokenKindLink)
		wantCode(t, f.deals.VerifyHandover(ctx, f.buyer, tr.ID, tokenCred(tok)), service.CodeInvalidHandoverToken)
	})

	t.Run("有効期限", func(t *testing.T) {
		// QR は5分、リンク・コードは24時間
		f.clock.Advance(5 * time.Minute)
		wantCode(t, f.deals.VerifyHandover(ctx, f.buyer, tr.ID, tokenCred(qr)), service.CodeInvalidHandoverToken)
		wantCode(t, f.deals.VerifyHandover(ctx, f.buyer, tr.ID, codeCred(qr)), service.CodeInvalidHandoverToken)
		if err := f.deals.VerifyHandover(ctx, f.buyer, tr.ID, tokenCred(link)); err != nil {
			t.Errorf("リンク: %v", err)
		}
		f.clock.Advance(24*time.Hour - 5*time.Minute)
		wantCode(t, f.deals.VerifyHandover(ctx, f.buyer, tr.ID, tokenCred(link)), service.CodeInvalidHandoverToken)
		wantCode(t, f.deals.VerifyHandover(ctx, f.buyer, tr.ID, codeCred(code)), service.CodeInvalidHandoverToken)
	})
}

// wrongCode2 は2つのコードのどちらとも違う6桁のコードを返す。
func wrongCode2(a, b string) string {
	for _, c := range []string{"000000", "000001", "000002"} {
		if c != a && c != b {
			return c
		}
	}
	panic("unreachable")
}

func TestDealComplete(t *testing.T) {
	ctx := context.Background()

	t.Run("購入者の操作で完了し、売上を確定する", func(t *testing.T) {
		tests := []struct {
			name   string
			kind   domain.HandoverTokenKind
			cred   func(*service.IssuedHandoverToken) service.HandoverCredential
			method domain.HandoverMethod
		}{
			{"QR", domain.HandoverTokenKindQR, tokenCred, domain.HandoverMethodQR},
			// QR の下の確認コードを入力したときは code（ADR 0024）
			{"QRの確認コード", domain.HandoverTokenKindQR, codeCred, domain.HandoverMethodCode},
			{"完了リンク", domain.HandoverTokenKindLink, tokenCred, domain.HandoverMethodLink},
			{"完了コード", domain.HandoverTokenKindCode, codeCred, domain.HandoverMethodCode},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				f := newDealFixture(t)
				tr := f.approved(t)
				tok := f.issue(t, tr.ID, tt.kind)
				f.tick()
				v, err := f.deals.Complete(ctx, f.buyer, tr.ID, tt.cred(tok))
				if err != nil {
					t.Fatal(err)
				}
				if v.Status != domain.TransactionStatusCompleted || v.CompletedAt == nil || !v.CompletedAt.Equal(f.clock.Now()) ||
					v.HandoverMethod == nil || *v.HandoverMethod != tt.method || v.Fee != nil {
					t.Errorf("購入者から見た取引 = %+v", v.Transaction)
				}
				if got := f.itemStatus(t); got != domain.ItemStatusCompleted {
					t.Errorf("商品のステータス = %s", got)
				}
				// 売上は承認時に記録した ¥1,500 − 手数料 ¥75（PAY-2）
				ps, err := f.payments.ListByTransaction(ctx, tr.ID)
				if err != nil {
					t.Fatal(err)
				}
				wantKinds(t, f.paymentKinds(t, tr.ID), domain.PaymentKindAuthorize, domain.PaymentKindCapture, domain.PaymentKindSettle)
				if last := ps[len(ps)-1]; last.Amount != 1425 || last.PaymentMethod != nil {
					t.Errorf("売上の確定 = %+v", last)
				}
				// 一度使ったトークンは使えない
				_, err = f.deals.Complete(ctx, f.buyer, tr.ID, tt.cred(tok))
				wantCode(t, err, apperr.CodeInvalidStatusTransition)
				wantCode(t, f.deals.VerifyHandover(ctx, f.buyer, tr.ID, tt.cred(tok)), apperr.CodeInvalidStatusTransition)
				wantKinds(t, f.paymentKinds(t, tr.ID), domain.PaymentKindAuthorize, domain.PaymentKindCapture, domain.PaymentKindSettle)
				// 出品者からも完了が見える
				sv, err := f.deals.Get(ctx, f.seller, tr.ID)
				if err != nil {
					t.Fatal(err)
				}
				if sv.Status != domain.TransactionStatusCompleted || sv.HandoverMethod == nil {
					t.Errorf("出品者から見た取引 = %+v", sv.Transaction)
				}
			})
		}
	})

	t.Run("完了できない場合", func(t *testing.T) {
		tests := []struct {
			name string
			// prepare は完了の前に状態を作り、完了する人と入力を返す
			prepare func(t *testing.T, f *dealFixture, tr *service.TransactionView, tok *service.IssuedHandoverToken) (*domain.User, service.HandoverCredential)
			code    apperr.Code
		}{
			{"出品者", func(_ *testing.T, f *dealFixture, _ *service.TransactionView, tok *service.IssuedHandoverToken) (*domain.User, service.HandoverCredential) {
				return f.seller, tokenCred(tok)
			}, apperr.CodeNotFound},
			{"取引に関係しない人", func(_ *testing.T, f *dealFixture, _ *service.TransactionView, tok *service.IssuedHandoverToken) (*domain.User, service.HandoverCredential) {
				return f.stranger, tokenCred(tok)
			}, apperr.CodeNotFound},
			{"期限切れ", func(_ *testing.T, f *dealFixture, _ *service.TransactionView, tok *service.IssuedHandoverToken) (*domain.User, service.HandoverCredential) {
				f.clock.Advance(5 * time.Minute)
				return f.buyer, tokenCred(tok)
			}, service.CodeInvalidHandoverToken},
			{"発行し直した前のトークン", func(t *testing.T, f *dealFixture, tr *service.TransactionView, tok *service.IssuedHandoverToken) (*domain.User, service.HandoverCredential) {
				f.issue(t, tr.ID, domain.HandoverTokenKindQR)
				return f.buyer, tokenCred(tok)
			}, service.CodeInvalidHandoverToken},
			{"キャンセル済み", func(t *testing.T, f *dealFixture, tr *service.TransactionView, tok *service.IssuedHandoverToken) (*domain.User, service.HandoverCredential) {
				if err := f.txs.UpdateStatus(context.Background(), tr.ID, domain.TransactionStatusInDeal, domain.TransactionStatusCancelled, f.clock.Now()); err != nil {
					t.Fatal(err)
				}
				return f.buyer, tokenCred(tok)
			}, apperr.CodeInvalidStatusTransition},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				f := newDealFixture(t)
				tr := f.approved(t)
				tok := f.issue(t, tr.ID, domain.HandoverTokenKindQR)
				u, cred := tt.prepare(t, f, tr, tok)
				_, err := f.deals.Complete(ctx, u, tr.ID, cred)
				wantCode(t, err, tt.code)
				if got := f.itemStatus(t); got != domain.ItemStatusInDeal {
					t.Errorf("商品のステータス = %s", got)
				}
				wantKinds(t, f.paymentKinds(t, tr.ID), domain.PaymentKindAuthorize, domain.PaymentKindCapture)
			})
		}
	})

	t.Run("同じ取引を同時に完了しても完了は1回だけ", func(t *testing.T) {
		f := newDealFixture(t)
		tr := f.approved(t)
		qr := f.issue(t, tr.ID, domain.HandoverTokenKindQR)
		link := f.issue(t, tr.ID, domain.HandoverTokenKindLink)
		// 同じトークンでの二重完了と、別のトークンでの同時の完了の両方を試す
		creds := []service.HandoverCredential{tokenCred(qr), tokenCred(qr), codeCred(qr), tokenCred(link)}
		const n = 8
		var wg sync.WaitGroup
		errs := make([]error, n)
		for i := range n {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, errs[i] = f.deals.Complete(ctx, f.buyer, tr.ID, creds[i%len(creds)])
			}()
		}
		wg.Wait()
		ok := 0
		for _, err := range errs {
			switch apperr.CodeOf(err) {
			case "":
				ok++
			case apperr.CodeInvalidStatusTransition, service.CodeInvalidHandoverToken:
			default:
				t.Errorf("err = %v", err)
			}
		}
		if ok != 1 {
			t.Errorf("成功 = %d, want 1", ok)
		}
		wantKinds(t, f.paymentKinds(t, tr.ID), domain.PaymentKindAuthorize, domain.PaymentKindCapture, domain.PaymentKindSettle)
	})
}
