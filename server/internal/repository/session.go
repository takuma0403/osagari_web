package repository

import (
	"context"

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
