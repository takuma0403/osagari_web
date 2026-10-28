package service

import (
	"context"
	"errors"

	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/repository"
)

// 取引成立後のキャンセル（双方の同意）と返金（DEAL-4, JOBS-2）。
// キャンセルの回数は列に持たず、キャンセルになった取引から数える（申請した人は同意日時の早い方）。

// Cancel は当事者が取引のキャンセルを申請する、または相手の申請に同意する（DEAL-4）。
// 双方の同意がそろったら取引をキャンセルにし、代金を購入者に返金して（モック。JOBS-2）、商品を公開中に戻す。
// 自分がすでに同意済みならそのまま返す。当事者でなければ 404、取引中でない（受け渡し完了後を含む）なら 409。
func (s *DealService) Cancel(ctx context.Context, u *domain.User, id string) (*TransactionView, error) {
	tr, role, err := s.findAsParty(ctx, u, id)
	if err != nil {
		return nil, err
	}
	if tr.Status != domain.TransactionStatusInDeal {
		return nil, errTransactionConflict
	}
	now := s.d.Clock.Now()
	err = s.d.Tx.Do(ctx, func(ctx context.Context) error {
		if err := s.d.Transactions.AgreeCancel(ctx, tr.ID, u.ID, role, now); err != nil {
			return err
		}
		err := s.d.Transactions.Cancel(ctx, tr.ID, now)
		// 相手がまだ同意していなければ、取引中のまま申請だけを記録する
		if errors.Is(err, repository.ErrConflict) {
			return nil
		}
		if err != nil {
			return err
		}
		return refundCancelled(ctx, s.d.Items, s.d.Payments, tr, domain.ItemStatusPublished, now)
	})
	if errors.Is(err, repository.ErrConflict) {
		// 二度押し・同時に押された場合は、最初の同意のまま返す
		v, err := s.view(ctx, u, tr.ID)
		if err != nil {
			return nil, err
		}
		if v.Status != domain.TransactionStatusInDeal || v.CancelAgreedAt(role) == nil {
			return nil, errTransactionConflict
		}
		return v, nil
	}
	if err := dealError(err); err != nil {
		return nil, err
	}
	return s.view(ctx, u, tr.ID)
}

// WithdrawCancel は当事者が自分の出したキャンセルの申請を取り下げる（DEAL-4）。
// 当事者でなければ 404、取引中でない（相手が同意してキャンセル済みを含む）・申請していないなら 409。
func (s *DealService) WithdrawCancel(ctx context.Context, u *domain.User, id string) (*TransactionView, error) {
	tr, role, err := s.findAsParty(ctx, u, id)
	if err != nil {
		return nil, err
	}
	err = s.d.Transactions.WithdrawCancel(ctx, tr.ID, u.ID, role, s.d.Clock.Now())
	if err := dealError(err); err != nil {
		return nil, err
	}
	return s.view(ctx, u, tr.ID)
}
