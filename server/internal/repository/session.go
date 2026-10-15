package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/takuma0403/osagari_web/server/internal/domain"
)

// sessionRow は sessions の1行（migrations/000002_create_sessions）。
type sessionRow struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt string
	RevokedAt *string
	CreatedAt string
	UpdatedAt string
}

func (sessionRow) TableName() string { return "sessions" }

// SessionRepository は sessions を読み書きする（AUTHN-5）。
type SessionRepository struct {
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// Create はセッションを保存する。ID が空なら採番する。
func (r *SessionRepository) Create(ctx context.Context, s *domain.Session) error {
	if s.ID == "" {
		s.ID = newID()
	}
	row := sessionRow{
		ID:        s.ID,
		UserID:    s.UserID,
		TokenHash: s.TokenHash,
		ExpiresAt: formatTime(s.ExpiresAt),
		RevokedAt: formatTimePtr(s.RevokedAt),
		CreatedAt: formatTime(s.CreatedAt),
		UpdatedAt: formatTime(s.UpdatedAt),
	}
	return conn(ctx, r.db).Create(&row).Error
}

func (r sessionRow) toDomain() (*domain.Session, error) {
	s := &domain.Session{ID: r.ID, UserID: r.UserID, TokenHash: r.TokenHash}
	var err error
	for _, f := range []struct {
		dst *time.Time
		src string
	}{{&s.ExpiresAt, r.ExpiresAt}, {&s.CreatedAt, r.CreatedAt}, {&s.UpdatedAt, r.UpdatedAt}} {
		if *f.dst, err = parseTime(f.src); err != nil {
			return nil, fmt.Errorf("repository: sessions: %w", err)
		}
	}
	if s.RevokedAt, err = parseTimePtr(r.RevokedAt); err != nil {
		return nil, fmt.Errorf("repository: sessions.revoked_at: %w", err)
	}
	return s, nil
}

// FindActiveByTokenHash はハッシュが一致する、失効しておらず期限内のセッションを返す。なければ ErrNotFound。
func (r *SessionRepository) FindActiveByTokenHash(ctx context.Context, tokenHash string, now time.Time) (*domain.Session, error) {
	var rows []sessionRow
	err := conn(ctx, r.db).
		Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", tokenHash, formatTime(now)).
		Limit(1).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return rows[0].toDomain()
}

// Revoke はハッシュが一致するセッションを失効させる（ログアウト。AUTHN-5）。
// 条件付きUPDATE で、まだ失効していないセッションだけを更新する。対象がなければ ErrNotFound。
func (r *SessionRepository) Revoke(ctx context.Context, tokenHash string, now time.Time) error {
	ts := formatTime(now)
	res := conn(ctx, r.db).Model(&sessionRow{}).
		Where("token_hash = ? AND revoked_at IS NULL", tokenHash).
		Updates(map[string]any{"revoked_at": ts, "updated_at": ts})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// RevokeAllByUser はユーザーのまだ失効していないセッションをすべて失効させる（パスワードの再設定。AUTHN-4）。
// 対象がなくてもエラーにしない。
func (r *SessionRepository) RevokeAllByUser(ctx context.Context, userID string, now time.Time) error {
	ts := formatTime(now)
	return conn(ctx, r.db).Model(&sessionRow{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Updates(map[string]any{"revoked_at": ts, "updated_at": ts}).Error
}

// trustedDeviceRow は trusted_devices の1行（migrations/000004_create_trusted_devices）。
type trustedDeviceRow struct {
	ID         string
	UserID     string
	DeviceHash string
	LastUsedAt string
	CreatedAt  string
	UpdatedAt  string
}

func (trustedDeviceRow) TableName() string { return "trusted_devices" }

// TrustedDeviceRepository は trusted_devices を読み書きする（AUTHN-3）。
type TrustedDeviceRepository struct {
	db *gorm.DB
}

func NewTrustedDeviceRepository(db *gorm.DB) *TrustedDeviceRepository {
	return &TrustedDeviceRepository{db: db}
}

// Upsert は端末を登録する。同じユーザーの同じ端末が既にあれば最終利用日時だけを更新する。
func (r *TrustedDeviceRepository) Upsert(ctx context.Context, d *domain.TrustedDevice) error {
	if d.ID == "" {
		d.ID = newID()
	}
	row := trustedDeviceRow{
		ID:         d.ID,
		UserID:     d.UserID,
		DeviceHash: d.DeviceHash,
		LastUsedAt: formatTime(d.LastUsedAt),
		CreatedAt:  formatTime(d.CreatedAt),
		UpdatedAt:  formatTime(d.UpdatedAt),
	}
	return conn(ctx, r.db).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "device_hash"}},
		DoUpdates: clause.AssignmentColumns([]string{"last_used_at", "updated_at"}),
	}).Create(&row).Error
}

// Exists はユーザーの端末が登録済みかを返す（AUTHN-3）。
func (r *TrustedDeviceRepository) Exists(ctx context.Context, userID, deviceHash string) (bool, error) {
	var n int64
	err := conn(ctx, r.db).Model(&trustedDeviceRow{}).
		Where("user_id = ? AND device_hash = ?", userID, deviceHash).Count(&n).Error
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
