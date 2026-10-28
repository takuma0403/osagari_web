package service

import (
	"context"
	"errors"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/clock"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/payment"
	"github.com/takuma0403/osagari_web/server/internal/repository"
)

var (
	errUserNotFound      = apperr.NotFound("ユーザーが見つかりません")
	errUserConflict      = apperr.InvalidStatusTransition("このユーザーの状態ではこの操作はできません")
	errExpiresOnInPast   = apperr.BadRequest("有効期限は今日以降の日付にしてください")
	errStudentNumberTerm = apperr.BadRequest("学籍番号は数字で入力してください")
)

// AdminUserDeps は AdminUserService の依存。
type AdminUserDeps struct {
	Tx           Transactor
	Users        AdminUserStore
	StudentCards RetainedStudentCardStore
	Items        ItemStore
	Transactions TransactionStore
	Sessions     SessionRevoker
	Payments     payment.Gateway
	Admins       domain.AdminList
	Clock        clock.Clock
}

// AdminUserService は管理者によるユーザーの検索・停止・停止の解除・有効期限の変更・退学による削除を行う（ADMIN, AUTH-3, ADR 0034）。
// どの操作も管理者でなければ 404 にする（ADR 0015）。
type AdminUserService struct {
	d AdminUserDeps
}

func NewAdminUserService(d AdminUserDeps) *AdminUserService {
	return &AdminUserService{d: d}
}

// AdminUserView は管理者に見せるユーザー。
type AdminUserView struct {
	*domain.User
	// CancelCount はキャンセル回数（双方の同意でキャンセルになった取引の数。DEAL-4, ADR 0034）
	CancelCount int
	// ActiveTransactionCount は当事者になっている取引中の取引の数
	ActiveTransactionCount int
}

// AdminUserListQuery は管理画面のユーザーの一覧の入力。
type AdminUserListQuery struct {
	// Q は学籍番号の前方一致。nil か空なら絞り込まない
	Q *string
	// Filter は状態での絞り込み。nil ならすべて
	Filter *domain.AdminUserFilter
	AdminPageQuery
}

// AdminUserPage は管理画面のユーザーの一覧の1ページ。
type AdminUserPage struct {
	Entries []*AdminUserView
	// WithdrawalRequestCount は退学を申請して削除されていない人の数（絞り込みに関係なく数える）
	WithdrawalRequestCount int
	NextCursor             *string
}

// List はユーザーを登録の新しい順に返す。条件が不正なら 400 bad_request。
func (s *AdminUserService) List(ctx context.Context, admin *domain.User, q AdminUserListQuery) (*AdminUserPage, error) {
	if err := requireAdmin(s.d.Admins, admin); err != nil {
		return nil, err
	}
	f := domain.AdminUserQuery{Filter: domain.AdminUserFilterAll}
	if q.Q != nil && *q.Q != "" {
		if !domain.ValidStudentNumberPrefix(*q.Q) {
			return nil, errStudentNumberTerm
		}
		f.StudentNumberPrefix = *q.Q
	}
	if q.Filter != nil {
		if !q.Filter.Valid() {
			return nil, errAdminListQuery
		}
		f.Filter = *q.Filter
	}
	after, limit, err := q.parse()
	if err != nil {
		return nil, err
	}
	users, err := s.d.Users.ListForAdmin(ctx, f, after, limit+1)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	users, next := adminPage(users, limit, func(u *domain.User) (time.Time, string) { return u.CreatedAt, u.ID })
	n, err := s.d.Users.CountWithdrawalRequested(ctx)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	views, err := s.views(ctx, users)
	if err != nil {
		return nil, err
	}
	return &AdminUserPage{Entries: views, WithdrawalRequestCount: n, NextCursor: next}, nil
}

// Get はユーザー1人を返す。いなければ 404。
func (s *AdminUserService) Get(ctx context.Context, admin *domain.User, id string) (*AdminUserView, error) {
	if err := requireAdmin(s.d.Admins, admin); err != nil {
		return nil, err
	}
	return s.view(ctx, id)
}

// Suspend は有効なアカウントを停止にし、セッションをすべて無効にする。商品・取引はそのまま（ADR 0034）。有効でなければ 409。
func (s *AdminUserService) Suspend(ctx context.Context, admin *domain.User, id string) (*AdminUserView, error) {
	return s.change(ctx, admin, id, func(ctx context.Context, u *domain.User, now time.Time) error {
		if err := s.d.Users.UpdateStatus(ctx, u.ID, []domain.UserStatus{domain.UserStatusActive}, domain.UserStatusSuspended, now); err != nil {
			return err
		}
		return s.d.Sessions.RevokeAllByUser(ctx, u.ID, now)
	})
}

// Unsuspend は停止中のアカウントを有効に戻す（ADR 0034）。停止中でなければ 409。
func (s *AdminUserService) Unsuspend(ctx context.Context, admin *domain.User, id string) (*AdminUserView, error) {
	return s.change(ctx, admin, id, func(ctx context.Context, u *domain.User, now time.Time) error {
		return s.d.Users.UpdateStatus(ctx, u.ID, []domain.UserStatus{domain.UserStatusSuspended}, domain.UserStatusActive, now)
	})
}

// UpdateExpiresOn は有効・停止のアカウントの有効期限を変える（延長の仕組みができるまでの暫定。AUTH-3, ADR 0034）。
// 今日（日本時間）より前の日付なら 400、有効・停止でなければ 409。
func (s *AdminUserService) UpdateExpiresOn(ctx context.Context, admin *domain.User, id string, expiresOn time.Time) (*AdminUserView, error) {
	return s.change(ctx, admin, id, func(ctx context.Context, u *domain.User, now time.Time) error {
		if expiresOn.Before(domain.JSTDate(now)) {
			return errExpiresOnInPast
		}
		return s.d.Users.UpdateExpiresOn(ctx, u.ID, domain.AdminClosableUserStatuses, expiresOn, now)
	})
}

// Delete は退学により有効・停止のアカウントを削除にし、失効と同じ後処理を行う（AUTH-3, ADR 0033, ADR 0034）。
// 削除日は操作した日（日本時間）。有効・停止でなければ、または後処理が並行した操作と競合したら 409。
func (s *AdminUserService) Delete(ctx context.Context, admin *domain.User, id string) (*AdminUserView, error) {
	return s.change(ctx, admin, id, func(ctx context.Context, u *domain.User, now time.Time) error {
		if err := s.d.Users.UpdateStatus(ctx, u.ID, domain.AdminClosableUserStatuses, domain.UserStatusDeleted, now); err != nil {
			return err
		}
		return accountCloser{
			StudentCards: s.d.StudentCards, Items: s.d.Items, Transactions: s.d.Transactions, Sessions: s.d.Sessions, Payments: s.d.Payments,
		}.close(ctx, u, domain.JSTDate(now), now)
	})
}

// change は管理者の確認とユーザーの読み出しの後、fn をトランザクションで行い、変更後のユーザーを返す。
// 条件付きUPDATE の競合は 409 にする。
func (s *AdminUserService) change(ctx context.Context, admin *domain.User, id string, fn func(ctx context.Context, u *domain.User, now time.Time) error) (*AdminUserView, error) {
	if err := requireAdmin(s.d.Admins, admin); err != nil {
		return nil, err
	}
	u, err := s.find(ctx, id)
	if err != nil {
		return nil, err
	}
	now := s.d.Clock.Now()
	err = s.d.Tx.Do(ctx, func(ctx context.Context) error { return fn(ctx, u, now) })
	var ae *apperr.Error
	switch {
	case errors.Is(err, repository.ErrConflict):
		return nil, errUserConflict
	case errors.As(err, &ae):
		return nil, err
	case err != nil:
		return nil, apperr.Internal(err)
	}
	return s.view(ctx, u.ID)
}

func (s *AdminUserService) find(ctx context.Context, id string) (*domain.User, error) {
	u, err := s.d.Users.FindByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, errUserNotFound
	}
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return u, nil
}

func (s *AdminUserService) view(ctx context.Context, id string) (*AdminUserView, error) {
	u, err := s.find(ctx, id)
	if err != nil {
		return nil, err
	}
	views, err := s.views(ctx, []*domain.User{u})
	if err != nil {
		return nil, err
	}
	return views[0], nil
}

// views はユーザーにキャンセル回数と取引中の件数を付ける。
func (s *AdminUserService) views(ctx context.Context, users []*domain.User) ([]*AdminUserView, error) {
	ids := make([]string, len(users))
	for i, u := range users {
		ids[i] = u.ID
	}
	cancels, err := s.d.Transactions.CountCancelledByUsers(ctx, ids)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	deals, err := s.d.Transactions.CountInDealByUsers(ctx, ids)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	out := make([]*AdminUserView, len(users))
	for i, u := range users {
		out[i] = &AdminUserView{User: u, CancelCount: cancels[u.ID], ActiveTransactionCount: deals[u.ID]}
	}
	return out, nil
}
