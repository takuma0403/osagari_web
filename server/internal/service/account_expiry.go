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
	"github.com/takuma0403/osagari_web/server/internal/storage"
)

// ExpiringUserStore は有効期限を過ぎたユーザーの読み出しと失効（repository.UserRepository）。
type ExpiringUserStore interface {
	ListToExpire(ctx context.Context, today time.Time) ([]*domain.User, error)
	Expire(ctx context.Context, id string, today, now time.Time) error
}

// RetainedStudentCardStore は学生証の削除予定日と画像の削除の記録（repository.StudentCardRepository）。
type RetainedStudentCardStore interface {
	SetDeleteAfter(ctx context.Context, userID string, date, now time.Time) (int, error)
	ListImagesToDelete(ctx context.Context, today time.Time) ([]*domain.StudentCard, error)
	MarkImageDeleted(ctx context.Context, id string, now time.Time) error
	ExistsByObjectKey(ctx context.Context, objectKey string) (bool, error)
}

// SessionRevoker はユーザーのセッションをすべて無効にする（repository.SessionRepository）。
type SessionRevoker interface {
	RevokeAllByUser(ctx context.Context, userID string, now time.Time) error
}

// AccountExpiryDeps は AccountExpiryService の依存。
type AccountExpiryDeps struct {
	Tx           Transactor
	Users        ExpiringUserStore
	StudentCards RetainedStudentCardStore
	Items        ItemStore
	Transactions TransactionStore
	Sessions     SessionRevoker
	Payments     payment.Gateway
	Storage      storage.Storage
	// Bucket は学生証の非公開バケットの名前
	Bucket string
	Clock  clock.Clock
}

// errAlreadyExpired は、一覧で読んだ後に別の処理で失効・削除されたユーザーを表す（失効の対象から外す）。
var errAlreadyExpired = errors.New("service: すでに失効・削除しています")

// AccountExpiryService はアカウントの失効と学生証画像の削除を行う。日次ジョブ（JOBS-1, ADR 0033）から呼ぶ。
type AccountExpiryService struct {
	d AccountExpiryDeps
}

func NewAccountExpiryService(d AccountExpiryDeps) *AccountExpiryService {
	return &AccountExpiryService{d: d}
}

// ExpireAccounts は有効期限（日本時間）を過ぎた有効・停止のアカウントを失効させ、失効させた人数を返す（AUTH-3, ADR 0033）。
// 1人ずつトランザクションで行い、その間に失効・削除された人は飛ばす（何度呼んでも結果は同じ）。1人が失敗しても残りは続け、失敗をまとめて返す。
// 商品・取引の更新が並行した操作と競合したときは、その人の失効を取り消して失敗として返す（Cloud Scheduler が再試行する）。
func (s *AccountExpiryService) ExpireAccounts(ctx context.Context) (int, error) {
	now := s.d.Clock.Now()
	today := domain.JSTDate(now)
	users, err := s.d.Users.ListToExpire(ctx, today)
	if err != nil {
		return 0, apperr.Internal(err)
	}
	n := 0
	var errs []error
	for _, u := range users {
		err := s.d.Tx.Do(ctx, func(ctx context.Context) error {
			return s.expire(ctx, u, today, now)
		})
		switch {
		case errors.Is(err, errAlreadyExpired):
		case err != nil:
			errs = append(errs, err)
		default:
			n++
		}
	}
	if err := errors.Join(errs...); err != nil {
		return n, apperr.Internal(err)
	}
	return n, nil
}

// expire は1人のアカウントを失効させる。トランザクションの中で呼ぶ。すでに失効・削除していれば errAlreadyExpired。
func (s *AccountExpiryService) expire(ctx context.Context, u *domain.User, today, now time.Time) error {
	err := s.d.Users.Expire(ctx, u.ID, today, now)
	if errors.Is(err, repository.ErrConflict) {
		return errAlreadyExpired
	}
	if err != nil {
		return err
	}
	return accountCloser{
		StudentCards: s.d.StudentCards, Items: s.d.Items, Transactions: s.d.Transactions, Sessions: s.d.Sessions, Payments: s.d.Payments,
	}.close(ctx, u, u.ExpiredOn(), now)
}

// DeleteStudentCardImages は削除予定日（日本時間の日付）を過ぎた学生証画像を消し、消した件数を返す（CARD, JOBS-1）。
// 行と閲覧の記録は残し、画像を消した日時だけを入れる。画像がすでになければ削除済みとして記録する。
// 1件が失敗しても残りは続け、失敗をまとめて返す。
func (s *AccountExpiryService) DeleteStudentCardImages(ctx context.Context) (int, error) {
	now := s.d.Clock.Now()
	cards, err := s.d.StudentCards.ListImagesToDelete(ctx, domain.JSTDate(now))
	if err != nil {
		return 0, apperr.Internal(err)
	}
	n := 0
	var errs []error
	for _, c := range cards {
		if err := s.d.Storage.Delete(ctx, s.d.Bucket, c.ObjectKey); err != nil && !errors.Is(err, storage.ErrNotFound) {
			errs = append(errs, err)
			continue
		}
		err := s.d.StudentCards.MarkImageDeleted(ctx, c.ID, now)
		switch {
		case errors.Is(err, repository.ErrConflict):
		case err != nil:
			errs = append(errs, err)
		default:
			n++
		}
	}
	if err := errors.Join(errs...); err != nil {
		return n, apperr.Internal(err)
	}
	return n, nil
}

// DeleteUnregisteredStudentCardImages は、アップロードされてから domain.UnregisteredUploadRetention たっても
// 学生証として登録されなかったオブジェクトを消し、消した件数を返す（STORAGE, ADR 0033）。
func (s *AccountExpiryService) DeleteUnregisteredStudentCardImages(ctx context.Context) (int, error) {
	before := s.d.Clock.Now().Add(-domain.UnregisteredUploadRetention)
	objects, err := s.d.Storage.List(ctx, s.d.Bucket)
	if err != nil {
		return 0, apperr.Internal(err)
	}
	n := 0
	var errs []error
	for _, o := range objects {
		if o.Created.After(before) {
			continue
		}
		registered, err := s.d.StudentCards.ExistsByObjectKey(ctx, o.Name)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if registered {
			continue
		}
		err = s.d.Storage.Delete(ctx, s.d.Bucket, o.Name)
		switch {
		case errors.Is(err, storage.ErrNotFound):
		case err != nil:
			errs = append(errs, err)
		default:
			n++
		}
	}
	if err := errors.Join(errs...); err != nil {
		return n, apperr.Internal(err)
	}
	return n, nil
}
