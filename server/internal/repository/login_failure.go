package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// loginFailureRow は login_failures の1行（migrations/000007_create_login_failures）。
type loginFailureRow struct {
	Email          string
	FailedAttempts int
	LockedUntil    *string
	CreatedAt      string
	UpdatedAt      string
}

func (loginFailureRow) TableName() string { return "login_failures" }

// LoginFailureRepository はパスワードの入力ミスの回数とログインの停止を読み書きする（AUTHN-6, ADR 0011）。
// 未登録のアドレスも同じように扱うため、ユーザーではなくメールアドレスで持つ。
type LoginFailureRepository struct {
	db *gorm.DB
}

func NewLoginFailureRepository(db *gorm.DB) *LoginFailureRepository {
	return &LoginFailureRepository{db: db}
}

// ReserveAttempt はパスワードを照合する前に入力の回数を1つ増やす（AUTHN-6）。
// 照合より先に数えることで、同時に大量の入力を送られても照合できるのは maxAttempts 回までになる。
// maxAttempts 回目を数えるときは、同じ文で now+lockFor までログインを止める（照合の後に止めると、
// その間にエラーや中断があったとき期限のないまま止まり続けるため）。その回の照合で一致したら Clear で解除する。
// 止めている期間が過ぎていれば、回数を0に戻してから数える。止めている間は数えずに ErrRateLimited を返す。
func (r *LoginFailureRepository) ReserveAttempt(ctx context.Context, email string, maxAttempts int, lockFor time.Duration, now time.Time) error {
	db := conn(ctx, r.db)
	ts := formatTime(now)
	if err := db.Exec(`INSERT INTO login_failures (email, created_at, updated_at) VALUES (?, ?, ?)
		ON CONFLICT (email) DO NOTHING`, email, ts, ts).Error; err != nil {
		return err
	}
	err := db.Model(&loginFailureRow{}).
		Where("email = ? AND locked_until IS NOT NULL AND locked_until <= ?", email, ts).
		Updates(map[string]any{"failed_attempts": 0, "locked_until": nil, "updated_at": ts}).Error
	if err != nil {
		return err
	}
	res := db.Model(&loginFailureRow{}).
		Where("email = ? AND locked_until IS NULL AND failed_attempts < ?", email, maxAttempts).
		Updates(map[string]any{
			"failed_attempts": gorm.Expr("failed_attempts + 1"),
			"locked_until":    gorm.Expr("CASE WHEN failed_attempts + 1 >= ? THEN ? ELSE NULL END", maxAttempts, formatTime(now.Add(lockFor))),
			"updated_at":      ts,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrRateLimited
	}
	return nil
}

// Clear はパスワードが一致したときに、入力の回数を0に戻し、止めていれば解除する。
// 止めている間は照合しないので、解除するのは上限の回目の照合で一致したときだけ。
func (r *LoginFailureRepository) Clear(ctx context.Context, email string, now time.Time) error {
	return conn(ctx, r.db).Model(&loginFailureRow{}).
		Where("email = ? AND (failed_attempts > 0 OR locked_until IS NOT NULL)", email).
		Updates(map[string]any{"failed_attempts": 0, "locked_until": nil, "updated_at": formatTime(now)}).Error
}

// Reset は入力の回数を0に戻す。止めている期間は変えない（パスワード再設定の後もロックは期限まで続く。ADR 0011）。
func (r *LoginFailureRepository) Reset(ctx context.Context, email string, now time.Time) error {
	return conn(ctx, r.db).Model(&loginFailureRow{}).
		Where("email = ? AND failed_attempts > 0", email).
		Updates(map[string]any{"failed_attempts": 0, "updated_at": formatTime(now)}).Error
}
