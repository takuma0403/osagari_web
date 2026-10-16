package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/takuma0403/osagari_web/server/internal/domain"
)

// verificationCodeRow は verification_codes の1行（migrations/000003_create_verification_codes）。
type verificationCodeRow struct {
	ID             string
	Email          string
	Purpose        string
	CodeHash       string
	FailedAttempts int
	ExpiresAt      string
	ConsumedAt     *string
	CreatedAt      string
	UpdatedAt      string
}

func (verificationCodeRow) TableName() string { return "verification_codes" }

func (r verificationCodeRow) toDomain() (*domain.VerificationCode, error) {
	c := &domain.VerificationCode{
		ID:             r.ID,
		Email:          r.Email,
		Purpose:        domain.VerificationPurpose(r.Purpose),
		CodeHash:       r.CodeHash,
		FailedAttempts: r.FailedAttempts,
	}
	var err error
	for _, f := range []struct {
		dst *time.Time
		src string
	}{{&c.ExpiresAt, r.ExpiresAt}, {&c.CreatedAt, r.CreatedAt}, {&c.UpdatedAt, r.UpdatedAt}} {
		if *f.dst, err = parseTime(f.src); err != nil {
			return nil, fmt.Errorf("repository: verification_codes: %w", err)
		}
	}
	if c.ConsumedAt, err = parseTimePtr(r.ConsumedAt); err != nil {
		return nil, fmt.Errorf("repository: verification_codes.consumed_at: %w", err)
	}
	return c, nil
}

// VerificationCodeRepository は verification_codes を読み書きする（AUTHN-2〜4, AUTHN-6）。
//
// 「使えるコード」は、未使用（consumed_at IS NULL）・期限内（expires_at > now）・入力の回数が上限未満のもの。
type VerificationCodeRepository struct {
	db *gorm.DB
}

func NewVerificationCodeRepository(db *gorm.DB) *VerificationCodeRepository {
	return &VerificationCodeRepository{db: db}
}

// Create はコードを保存する。ID が空なら採番する。
func (r *VerificationCodeRepository) Create(ctx context.Context, c *domain.VerificationCode) error {
	if c.ID == "" {
		c.ID = newID()
	}
	row := verificationCodeRow{
		ID:             c.ID,
		Email:          c.Email,
		Purpose:        string(c.Purpose),
		CodeHash:       c.CodeHash,
		FailedAttempts: c.FailedAttempts,
		ExpiresAt:      formatTime(c.ExpiresAt),
		ConsumedAt:     formatTimePtr(c.ConsumedAt),
		CreatedAt:      formatTime(c.CreatedAt),
		UpdatedAt:      formatTime(c.UpdatedAt),
	}
	return conn(ctx, r.db).Create(&row).Error
}

// CreateWithinLimit は、同じアドレスへの送信が上限の内側のときだけコードを保存する（AUTHN-6）。ID が空なら採番する。
// 用途をまとめて数え、直近 minInterval 以内に1件でもあるか、直近 window 以内に maxInWindow 件あれば保存せずに ErrRateLimited を返す。
// 判定と保存を1つの文で行うので、同時に送られても上限を超えない。c.CreatedAt を現在時刻として使う。
func (r *VerificationCodeRepository) CreateWithinLimit(ctx context.Context, c *domain.VerificationCode, minInterval, window time.Duration, maxInWindow int) error {
	if c.ID == "" {
		c.ID = newID()
	}
	res := conn(ctx, r.db).Exec(`INSERT INTO verification_codes
		(id, email, purpose, code_hash, failed_attempts, expires_at, consumed_at, created_at, updated_at)
		SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?
		WHERE NOT EXISTS (SELECT 1 FROM verification_codes WHERE email = ? AND created_at > ?)
		AND (SELECT COUNT(*) FROM verification_codes WHERE email = ? AND created_at > ?) < ?`,
		c.ID, c.Email, string(c.Purpose), c.CodeHash, c.FailedAttempts, formatTime(c.ExpiresAt),
		formatTimePtr(c.ConsumedAt), formatTime(c.CreatedAt), formatTime(c.UpdatedAt),
		c.Email, formatTime(c.CreatedAt.Add(-minInterval)),
		c.Email, formatTime(c.CreatedAt.Add(-window)), maxInWindow)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrRateLimited
	}
	return nil
}

// CountByEmailSince は、そのアドレスに since より後に保存したコードの数を用途をまとめて返す（AUTHN-6）。
func (r *VerificationCodeRepository) CountByEmailSince(ctx context.Context, email string, since time.Time) (int, error) {
	var n int64
	err := conn(ctx, r.db).Model(&verificationCodeRow{}).
		Where("email = ? AND created_at > ?", email, formatTime(since)).Count(&n).Error
	return int(n), err
}

// CountSince は、全アドレスに since より後に保存したコードの数（＝送ったコードのメールの数）を返す（AUTHN-6）。
func (r *VerificationCodeRepository) CountSince(ctx context.Context, since time.Time) (int, error) {
	var n int64
	err := conn(ctx, r.db).Model(&verificationCodeRow{}).
		Where("created_at > ?", formatTime(since)).Count(&n).Error
	return int(n), err
}

// InvalidateOthers は同じアドレス・用途の未使用のコードのうち、keepID 以外をすべて無効にする（再送したら古いコードは使えなくする）。
func (r *VerificationCodeRepository) InvalidateOthers(ctx context.Context, email string, purpose domain.VerificationPurpose, keepID string, now time.Time) error {
	ts := formatTime(now)
	return conn(ctx, r.db).Model(&verificationCodeRow{}).
		Where("email = ? AND purpose = ? AND consumed_at IS NULL AND id <> ?", email, string(purpose), keepID).
		Updates(map[string]any{"consumed_at": ts, "updated_at": ts}).Error
}

// FindLatestActive は同じアドレス・用途の、いちばん新しい使えるコードを返す。なければ ErrNotFound。
func (r *VerificationCodeRepository) FindLatestActive(ctx context.Context, email string, purpose domain.VerificationPurpose, maxAttempts int, now time.Time) (*domain.VerificationCode, error) {
	var rows []verificationCodeRow
	err := conn(ctx, r.db).
		Where("email = ? AND purpose = ? AND consumed_at IS NULL AND expires_at > ? AND failed_attempts < ?",
			email, string(purpose), formatTime(now), maxAttempts).
		Order("created_at DESC").Limit(1).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return rows[0].toDomain()
}

// ReserveAttempt は照合の前に入力の回数を1つ増やし、増やした後の回数を返す（AUTHN-6）。
// 照合より先に数えることで、同時に大量の入力を送られても1つのコードを照合できるのは maxAttempts 回までになる。
// 照合に成功したら Consume で使用済みにするので、成功した回が数に入っても問題ない。
// コードが使えない（使用済み・期限切れ・回数が上限）なら ErrConflict。
func (r *VerificationCodeRepository) ReserveAttempt(ctx context.Context, id string, maxAttempts int, now time.Time) (int, error) {
	db := conn(ctx, r.db)
	ts := formatTime(now)
	res := db.Model(&verificationCodeRow{}).
		Where("id = ? AND consumed_at IS NULL AND expires_at > ? AND failed_attempts < ?", id, ts, maxAttempts).
		Updates(map[string]any{"failed_attempts": gorm.Expr("failed_attempts + 1"), "updated_at": ts})
	if res.Error != nil {
		return 0, res.Error
	}
	if res.RowsAffected == 0 {
		return 0, ErrConflict
	}
	var attempts int
	if err := db.Model(&verificationCodeRow{}).Where("id = ?", id).Select("failed_attempts").Scan(&attempts).Error; err != nil {
		return 0, err
	}
	return attempts, nil
}

// Consume はコードを使用済みにする。条件付きUPDATE なので一度しか成功しない。使えないコードなら ErrConflict。
func (r *VerificationCodeRepository) Consume(ctx context.Context, id string, now time.Time) error {
	ts := formatTime(now)
	res := conn(ctx, r.db).Model(&verificationCodeRow{}).
		Where("id = ? AND consumed_at IS NULL AND expires_at > ?", id, ts).
		Updates(map[string]any{"consumed_at": ts, "updated_at": ts})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrConflict
	}
	return nil
}
