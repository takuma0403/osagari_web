package service

import (
	"context"
	"errors"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/payment"
	"github.com/takuma0403/osagari_web/server/internal/repository"
)

// accountCloser は失効・削除したアカウントの後処理を行う（AUTH-3, ADR 0033, ADR 0034）。
type accountCloser struct {
	StudentCards RetainedStudentCardStore
	Items        ItemStore
	Transactions TransactionStore
	Sessions     SessionRevoker
	Payments     payment.Gateway
}

// close は失効・削除した u の後処理を行う。トランザクションの中で、u の状態を変えた後に呼ぶ。closedOn は失効日・削除日。
// 商品の取り下げ、購入希望の期限切れ、取引中の取引のキャンセルと返金、学生証の削除予定日の記録、セッションの無効化を行う。
func (c accountCloser) close(ctx context.Context, u *domain.User, closedOn, now time.Time) error {
	// 学生証は再提出で置き換えたものも含めて、失効・削除日から3年で消す（ADR 0014）
	if _, err := c.StudentCards.SetDeleteAfter(ctx, u.ID, domain.StudentCardDeleteAfter(closedOn), now); err != nil {
		return err
	}
	// 本人が出した、または本人の商品への承認待ちの購入希望は期限切れにして枠を解放する
	reqs, err := c.Transactions.ListByPartyAndStatus(ctx, u.ID, domain.TransactionStatusRequested)
	if err != nil {
		return err
	}
	for _, tr := range reqs {
		if err := endRequest(ctx, c.Transactions, c.Items, c.Payments, tr, domain.TransactionStatusExpired, now); err != nil {
			return err
		}
	}
	// 取引中の取引は同意なしでキャンセルして返金する（DEAL-4 の例外）。商品は、u が出品者なら取り下げ、購入者なら公開中に戻す
	deals, err := c.Transactions.ListByPartyAndStatus(ctx, u.ID, domain.TransactionStatusInDeal)
	if err != nil {
		return err
	}
	for _, tr := range deals {
		if err := c.Transactions.CancelBySystem(ctx, tr.ID, now); err != nil {
			return err
		}
		to := domain.ItemStatusPublished
		if tr.SellerID == u.ID {
			to = domain.ItemStatusWithdrawn
		}
		if err := refundCancelled(ctx, c.Items, c.Payments, tr, to, now); err != nil {
			return err
		}
	}
	// 本人の商品を取り下げる。購入希望は上で期限切れにしたので残っていない
	items, err := c.Items.ListBySeller(ctx, u.ID)
	if err != nil {
		return err
	}
	for _, it := range items {
		if !it.CanWithdraw() {
			continue
		}
		if err := c.Items.UpdateStatus(ctx, it.ID, u.ID, domain.ItemWithdrawableStatuses, nil, domain.ItemStatusWithdrawn, now); err != nil {
			return err
		}
	}
	return c.Sessions.RevokeAllByUser(ctx, u.ID, now)
}

// refundCancelled はキャンセルにした取引の商品を取引中から itemTo に戻し、代金を購入者に返金する（DEAL-4, JOBS-2）。
// 商品が取引中でなければ（管理者の停止など）そのままにし、キャンセルと返金は止めない。トランザクションの中で呼ぶ。
func refundCancelled(ctx context.Context, items ItemStore, gw payment.Gateway, tr *domain.Transaction, itemTo domain.ItemStatus, now time.Time) error {
	err := items.UpdateDealStatus(ctx, tr.ItemID, tr.SellerID, []domain.ItemStatus{domain.ItemStatusInDeal}, itemTo, now)
	if err != nil && !errors.Is(err, repository.ErrConflict) {
		return err
	}
	return gw.Refund(ctx, tr)
}
