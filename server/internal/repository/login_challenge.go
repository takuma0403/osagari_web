package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/takuma0403/osagari_web/server/internal/domain"
)

// loginChallengeRow は login_challenges の1行（migrations/000006_create_login_challenges）。
type loginChallengeRow struct {
	ID         string
	UserID     string
	TokenHash  string
	Reason     string
	DeviceHash *string
	ExpiresAt  string
	ConsumedAt *string
	CreatedAt  string
	UpdatedAt  string
}

func (loginChallengeRow) TableName() string { return "login_challenges" }

func (r loginChallengeRow) toDomain() (*domain.LoginChallenge, error) {
	c := &domain.LoginChallenge{
		ID:        r.ID,
		UserID:    r.UserID,
		TokenHash: r.TokenHash,
		Reason:    domain.LoginRecheckReason(r.Reason),
	}
	if r.DeviceHash != nil {
		c.DeviceHash = *r.DeviceHash
	}
	var err error
	for _, f := range []struct {
		dst *time.Time
		src string
	}{{&c.ExpiresAt, r.ExpiresAt}, {&c.CreatedAt, r.CreatedAt}, {&c.UpdatedAt, r.UpdatedAt}} {
		if *f.dst, err = parseTime(f.src); err != nil {
			return nil, fmt.Errorf("repository: login_challenges: %w", err)
		}
	}
	if c.ConsumedAt, err = parseTimePtr(r.ConsumedAt); err != nil {
		return nil, fmt.Errorf("repository: login_challenges.consumed_at: %w", err)
	}
	return c, nil
}

// LoginChallengeRepository は login_challenges を読み書きする（AUTHN-3, ADR 0009）。
//
// 「使えるチャレンジ」は、未使用（consumed_at IS NULL）・期限内（expires_at > now）のもの。
type LoginChallengeRepository struct {
	db *gorm.DB
}

func NewLoginChallengeRepository(db *gorm.DB) *LoginChallengeRepository {
	return &LoginChallengeRepository{db: db}
}

// Create はチャレンジを保存する。ID が空なら採番する。DeviceHash が空なら NULL で保存する。
func (r *LoginChallengeRepository) Create(ctx context.Context, c *domain.LoginChallenge) error {
	if c.ID == "" {
		c.ID = newID()
	}
	row := loginChallengeRow{
		ID:         c.ID,
		UserID:     c.UserID,
		TokenHash:  c.TokenHash,
		Reason:     string(c.Reason),
		ExpiresAt:  formatTime(c.ExpiresAt),
		ConsumedAt: formatTimePtr(c.ConsumedAt),
		CreatedAt:  formatTime(c.CreatedAt),
		UpdatedAt:  formatTime(c.UpdatedAt),
	}
	if c.DeviceHash != "" {
		row.DeviceHash = &c.DeviceHash
	}
	return conn(ctx, r.db).Create(&row).Error
}

// FindActive はハッシュが一致する使えるチャレンジを返す。なければ ErrNotFound。
func (r *LoginChallengeRepository) FindActive(ctx context.Context, tokenHash string, now time.Time) (*domain.LoginChallenge, error) {
	var rows []loginChallengeRow
	err := conn(ctx, r.db).
		Where("token_hash = ? AND consumed_at IS NULL AND expires_at > ?", tokenHash, formatTime(now)).
		Limit(1).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return rows[0].toDomain()
}

// Consume はチャレンジを使用済みにする。条件付きUPDATE なので一度しか成功しない。使えないチャレンジなら ErrConflict。
func (r *LoginChallengeRepository) Consume(ctx context.Context, id string, now time.Time) error {
	ts := formatTime(now)
	res := conn(ctx, r.db).Model(&loginChallengeRow{}).
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
