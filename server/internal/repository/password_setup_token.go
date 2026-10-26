package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/takuma0403/osagari_web/server/internal/domain"
)

// passwordSetupTokenRow は password_setup_tokens の1行（migrations/000005_create_password_setup_tokens）。
type passwordSetupTokenRow struct {
	ID         string
	Email      string
	Purpose    string
	TokenHash  string
	ExpiresAt  string
	ConsumedAt *string
	CreatedAt  string
	UpdatedAt  string
}

func (passwordSetupTokenRow) TableName() string { return "password_setup_tokens" }

func (r passwordSetupTokenRow) toDomain() (*domain.PasswordSetupToken, error) {
	t := &domain.PasswordSetupToken{
		ID:        r.ID,
		Email:     r.Email,
		Purpose:   domain.PasswordSetupPurpose(r.Purpose),
		TokenHash: r.TokenHash,
	}
	var err error
	for _, f := range []struct {
		dst *time.Time
		src string
	}{{&t.ExpiresAt, r.ExpiresAt}, {&t.CreatedAt, r.CreatedAt}, {&t.UpdatedAt, r.UpdatedAt}} {
		if *f.dst, err = parseTime(f.src); err != nil {
			return nil, fmt.Errorf("repository: password_setup_tokens: %w", err)
		}
	}
	if t.ConsumedAt, err = parseTimePtr(r.ConsumedAt); err != nil {
		return nil, fmt.Errorf("repository: password_setup_tokens.consumed_at: %w", err)
	}
	return t, nil
}

// PasswordSetupTokenRepository は password_setup_tokens を読み書きする（AUTHN-2, AUTHN-4, ADR 0007）。
type PasswordSetupTokenRepository struct {
	db *gorm.DB
}

func NewPasswordSetupTokenRepository(db *gorm.DB) *PasswordSetupTokenRepository {
	return &PasswordSetupTokenRepository{db: db}
}

// Create は一時トークンを保存する。ID が空なら採番する。
func (r *PasswordSetupTokenRepository) Create(ctx context.Context, t *domain.PasswordSetupToken) error {
	if t.ID == "" {
		t.ID = newID()
	}
	row := passwordSetupTokenRow{
		ID:         t.ID,
		Email:      t.Email,
		Purpose:    string(t.Purpose),
		TokenHash:  t.TokenHash,
		ExpiresAt:  formatTime(t.ExpiresAt),
		ConsumedAt: formatTimePtr(t.ConsumedAt),
		CreatedAt:  formatTime(t.CreatedAt),
		UpdatedAt:  formatTime(t.UpdatedAt),
	}
	return conn(ctx, r.db).Create(&row).Error
}

// DeleteExpired は期限が切れたトークンを消し、消した件数を返す（JOBS-1）。
func (r *PasswordSetupTokenRepository) DeleteExpired(ctx context.Context, now time.Time) (int, error) {
	res := conn(ctx, r.db).Where("expires_at <= ?", formatTime(now)).Delete(&passwordSetupTokenRow{})
	return int(res.RowsAffected), res.Error
}

// Consume はハッシュが一致する未使用・期限内のトークンを使用済みにして返す。
// 条件付きUPDATE なので、同じトークンは一度しか使えない。使えないトークンなら ErrConflict。
func (r *PasswordSetupTokenRepository) Consume(ctx context.Context, tokenHash string, purpose domain.PasswordSetupPurpose, now time.Time) (*domain.PasswordSetupToken, error) {
	db := conn(ctx, r.db)
	ts := formatTime(now)
	res := db.Model(&passwordSetupTokenRow{}).
		Where("token_hash = ? AND purpose = ? AND consumed_at IS NULL AND expires_at > ?", tokenHash, string(purpose), ts).
		Updates(map[string]any{"consumed_at": ts, "updated_at": ts})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, ErrConflict
	}
	var rows []passwordSetupTokenRow
	if err := db.Where("token_hash = ?", tokenHash).Limit(1).Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return rows[0].toDomain()
}
