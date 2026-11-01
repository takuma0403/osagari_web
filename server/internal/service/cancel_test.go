package service_test

import (
	"context"
	"sync"
	"testing"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/service"
)

func TestDealCancel(t *testing.T) {
	ctx := context.Background()

	t.Run("双方の同意でキャンセルし、返金して商品を公開中に戻す", func(t *testing.T) {
		tests := []struct {
			name string
			// first は申請する人、second は同意する人
			first, second func(f *dealFixture) *domain.User
		}{
			{"購入者が申請", func(f *dealFixture) *domain.User { return f.buyer }, func(f *dealFixture) *domain.User { return f.seller }},
			{"出品者が申請", func(f *dealFixture) *domain.User { return f.seller }, func(f *dealFixture) *domain.User { return f.buyer }},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				f := newDealFixture(t)
				tr := f.approved(t)
				requestedAt := f.clock.Now()
				v, err := f.deals.Cancel(ctx, tt.first(f), tr.ID)
				if err != nil {
					t.Fatal(err)
				}
				// 片方だけではキャンセルにならない
				if v.Status != domain.TransactionStatusInDeal || v.CancelAgreedAt(v.Role) == nil || !v.CancelAgreedAt(v.Role).Equal(requestedAt) {
					t.Errorf("申請後の取引 = %+v", v.Transaction)
				}
				if got := f.itemStatus(t); got != domain.ItemStatusInDeal {
					t.Errorf("申請後の商品のステータス = %s", got)
				}
				wantKinds(t, f.paymentKinds(t, tr.ID), domain.PaymentKindAuthorize, domain.PaymentKindCapture)

				f.tick()
				v, err = f.deals.Cancel(ctx, tt.second(f), tr.ID)
				if err != nil {
					t.Fatal(err)
				}
				if v.Status != domain.TransactionStatusCancelled || v.BuyerCancelAgreedAt == nil || v.SellerCancelAgreedAt == nil {
					t.Errorf("同意後の取引 = %+v", v.Transaction)
				}
				if got := f.itemStatus(t); got != domain.ItemStatusPublished {
					t.Errorf("同意後の商品のステータス = %s", got)
				}
				// 返金は代金の全額（JOBS-2）
				wantKinds(t, f.paymentKinds(t, tr.ID), domain.PaymentKindAuthorize, domain.PaymentKindCapture, domain.PaymentKindRefund)
				ps, err := f.payments.ListByTransaction(ctx, tr.ID)
				if err != nil {
					t.Fatal(err)
				}
				if last := ps[len(ps)-1]; last.Amount != 1500 || last.PaymentMethod == nil || *last.PaymentMethod != domain.PaymentMethodCard {
					t.Errorf("返金 = %+v", last)
				}
				// キャンセル後は申請も取り下げもできない
				_, err = f.deals.Cancel(ctx, tt.first(f), tr.ID)
				wantCode(t, err, apperr.CodeInvalidStatusTransition)
				_, err = f.deals.WithdrawCancel(ctx, tt.first(f), tr.ID)
				wantCode(t, err, apperr.CodeInvalidStatusTransition)
				wantKinds(t, f.paymentKinds(t, tr.ID), domain.PaymentKindAuthorize, domain.PaymentKindCapture, domain.PaymentKindRefund)
			})
		}
	})

	t.Run("同意済みでもう一度申請したら最初の日時のまま返す", func(t *testing.T) {
		f := newDealFixture(t)
		tr := f.approved(t)
		first, err := f.deals.Cancel(ctx, f.buyer, tr.ID)
		if err != nil {
			t.Fatal(err)
		}
		f.tick()
		v, err := f.deals.Cancel(ctx, f.buyer, tr.ID)
		if err != nil {
			t.Fatal(err)
		}
		if v.Status != domain.TransactionStatusInDeal || !v.BuyerCancelAgreedAt.Equal(*first.BuyerCancelAgreedAt) || v.SellerCancelAgreedAt != nil {
			t.Errorf("取引 = %+v", v.Transaction)
		}
	})

	t.Run("キャンセルできない場合", func(t *testing.T) {
		tests := []struct {
			name string
			// prepare は取引を用意し、申請する人と取引IDを返す
			prepare func(t *testing.T, f *dealFixture) (*domain.User, string)
			code    apperr.Code
		}{
			{"取引に関係しない人", func(t *testing.T, f *dealFixture) (*domain.User, string) {
				return f.stranger, f.approved(t).ID
			}, apperr.CodeNotFound},
			{"ない取引", func(_ *testing.T, f *dealFixture) (*domain.User, string) {
				return f.buyer, "no-such-transaction"
			}, apperr.CodeNotFound},
			{"承認待ち", func(t *testing.T, f *dealFixture) (*domain.User, string) {
				return f.buyer, f.request(t, f.buyer).ID
			}, apperr.CodeInvalidStatusTransition},
			{"却下済み", func(t *testing.T, f *dealFixture) (*domain.User, string) {
				tr := f.request(t, f.buyer)
				if _, err := f.deals.Reject(context.Background(), f.seller, tr.ID); err != nil {
					t.Fatal(err)
				}
				return f.seller, tr.ID
			}, apperr.CodeInvalidStatusTransition},
			// DEAL-4: 受け渡し完了後はキャンセル・返金できない。申請中でも購入者は完了できる
			{"申請中に受け渡しを完了した後", func(t *testing.T, f *dealFixture) (*domain.User, string) {
				tr := f.approved(t)
				if _, err := f.deals.Cancel(context.Background(), f.seller, tr.ID); err != nil {
					t.Fatal(err)
				}
				tok := f.issue(t, tr.ID, domain.HandoverTokenKindQR)
				if _, err := f.deals.Complete(context.Background(), f.buyer, tr.ID, tokenCred(tok)); err != nil {
					t.Fatal(err)
				}
				return f.buyer, tr.ID
			}, apperr.CodeInvalidStatusTransition},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				f := newDealFixture(t)
				u, id := tt.prepare(t, f)
				_, err := f.deals.Cancel(ctx, u, id)
				wantCode(t, err, tt.code)
				for _, k := range f.paymentKinds(t, id) {
					if k == domain.PaymentKindRefund {
						t.Errorf("返金が記録された: %v", f.paymentKinds(t, id))
					}
				}
			})
		}
	})

	t.Run("双方が同時に同意してもキャンセル・返金は1回だけ", func(t *testing.T) {
		f := newDealFixture(t)
		tr := f.approved(t)
		users := []*domain.User{f.buyer, f.seller}
		const n = 8
		var wg sync.WaitGroup
		errs := make([]error, n)
		for i := range n {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, errs[i] = f.deals.Cancel(ctx, users[i%len(users)], tr.ID)
			}()
		}
		wg.Wait()
		for _, err := range errs {
			if c := apperr.CodeOf(err); c != "" && c != apperr.CodeInvalidStatusTransition {
				t.Errorf("err = %v", err)
			}
		}
		if got := f.txStatus(t, tr.ID); got != domain.TransactionStatusCancelled {
			t.Errorf("取引のステータス = %s", got)
		}
		wantKinds(t, f.paymentKinds(t, tr.ID), domain.PaymentKindAuthorize, domain.PaymentKindCapture, domain.PaymentKindRefund)
	})
}

func TestDealWithdrawCancel(t *testing.T) {
	ctx := context.Background()

	t.Run("自分の申請を取り下げる", func(t *testing.T) {
		f := newDealFixture(t)
		tr := f.approved(t)
		if _, err := f.deals.Cancel(ctx, f.seller, tr.ID); err != nil {
			t.Fatal(err)
		}
		v, err := f.deals.WithdrawCancel(ctx, f.seller, tr.ID)
		if err != nil {
			t.Fatal(err)
		}
		if v.Status != domain.TransactionStatusInDeal || v.SellerCancelAgreedAt != nil {
			t.Errorf("取引 = %+v", v.Transaction)
		}
		// 取り下げた後に相手が申請しても、片方だけなのでキャンセルにならない
		v, err = f.deals.Cancel(ctx, f.buyer, tr.ID)
		if err != nil {
			t.Fatal(err)
		}
		if v.Status != domain.TransactionStatusInDeal {
			t.Errorf("取り下げ後の相手の申請 = %+v", v.Transaction)
		}
		wantKinds(t, f.paymentKinds(t, tr.ID), domain.PaymentKindAuthorize, domain.PaymentKindCapture)
	})

	t.Run("取り下げられない場合", func(t *testing.T) {
		tests := []struct {
			name    string
			prepare func(t *testing.T, f *dealFixture, tr *service.TransactionView) *domain.User
			code    apperr.Code
		}{
			{"申請していない", func(_ *testing.T, f *dealFixture, _ *service.TransactionView) *domain.User {
				return f.buyer
			}, apperr.CodeInvalidStatusTransition},
			{"相手の申請は取り下げられない", func(t *testing.T, f *dealFixture, tr *service.TransactionView) *domain.User {
				if _, err := f.deals.Cancel(context.Background(), f.seller, tr.ID); err != nil {
					t.Fatal(err)
				}
				return f.buyer
			}, apperr.CodeInvalidStatusTransition},
			{"取引に関係しない人", func(t *testing.T, f *dealFixture, tr *service.TransactionView) *domain.User {
				if _, err := f.deals.Cancel(context.Background(), f.seller, tr.ID); err != nil {
					t.Fatal(err)
				}
				return f.stranger
			}, apperr.CodeNotFound},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				f := newDealFixture(t)
				tr := f.approved(t)
				u := tt.prepare(t, f, tr)
				_, err := f.deals.WithdrawCancel(ctx, u, tr.ID)
				wantCode(t, err, tt.code)
			})
		}
	})
}
