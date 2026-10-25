package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/takuma0403/osagari_web/server/internal/domain"
)

// handoverTokenRow は handover_tokens の1行（migrations/000015_create_handover_tokens）。
type handoverTokenRow struct {
	ID            string
	TransactionID string
	Kind          string
	TokenHash     *string
	CodeHash      *string
	ExpiresAt     string
	ConsumedAt    *string
	CreatedAt     string
	UpdatedAt     string
}

func (handoverTokenRow) TableName() string { return "handover_tokens" }

func (r handoverTokenRow) toDomain() (*domain.HandoverToken, error) {
	t := &domain.HandoverToken{
		ID:            r.ID,
		TransactionID: r.TransactionID,
		Kind:          domain.HandoverTokenKind(r.Kind),
		TokenHash:     r.TokenHash,
		CodeHash:      r.CodeHash,
	}
	var err error
	for _, f := range []struct {
		dst *time.Time
		src string
	}{{&t.ExpiresAt, r.ExpiresAt}, {&t.CreatedAt, r.CreatedAt}, {&t.UpdatedAt, r.UpdatedAt}} {
		if *f.dst, err = parseTime(f.src); err != nil {
			return nil, fmt.Errorf("repository: handover_tokens: %w", err)
		}
	}
	if t.ConsumedAt, err = parseTimePtr(r.ConsumedAt); err != nil {
		return nil, fmt.Errorf("repository: handover_tokens.consumed_at: %w", err)
	}
	return t, nil
}

// HandoverTokenRepository は受け渡しトークン（handover_tokens）を読み書きする（DEAL-3）。
type HandoverTokenRepository struct {
	db *gorm.DB
}

func NewHandoverTokenRepository(db *gorm.DB) *HandoverTokenRepository {
	return &HandoverTokenRepository{db: db}
}

// Create は受け渡しトークンを保存する。ID が空なら採番する。
func (r *HandoverTokenRepository) Create(ctx context.Context, t *domain.HandoverToken) error {
	if t.ID == "" {
		t.ID = newID()
	}
	return conn(ctx, r.db).Create(&handoverTokenRow{
		ID:            t.ID,
		TransactionID: t.TransactionID,
		Kind:          string(t.Kind),
		TokenHash:     t.TokenHash,
		CodeHash:      t.CodeHash,
		ExpiresAt:     formatTime(t.ExpiresAt),
		ConsumedAt:    formatTimePtr(t.ConsumedAt),
		CreatedAt:     formatTime(t.CreatedAt),
		UpdatedAt:     formatTime(t.UpdatedAt),
	}).Error
}

// InvalidateOthers は同じ取引・同じ種類の未使用のトークンのうち keepID 以外を使えなくする（使えるのは常に最新の1つ）。
func (r *HandoverTokenRepository) InvalidateOthers(ctx context.Context, transactionID string, kind domain.HandoverTokenKind, keepID string, now time.Time) error {
	ts := formatTime(now)
	return conn(ctx, r.db).Model(&handoverTokenRow{}).
		Where("transaction_id = ? AND kind = ? AND consumed_at IS NULL AND id <> ?", transactionID, string(kind), keepID).
		Updates(map[string]any{"consumed_at": ts, "updated_at": ts}).Error
}

// FindActiveByTokenHash はハッシュが一致する、取引の未使用・期限内のトークンを返す。なければ ErrNotFound。
func (r *HandoverTokenRepository) FindActiveByTokenHash(ctx context.Context, transactionID, tokenHash string, now time.Time) (*domain.HandoverToken, error) {
	var rows []handoverTokenRow
	err := conn(ctx, r.db).
		Where("token_hash = ? AND transaction_id = ? AND consumed_at IS NULL AND expires_at > ?", tokenHash, transactionID, formatTime(now)).
		Limit(1).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return rows[0].toDomain()
}

// ListActiveWithCode は取引の未使用・期限内の、6桁のコードを持つトークンを返す（コードの照合用）。
// コードはソルト付きのハッシュで検索できないため、候補を返して呼び出し側で照合する。
func (r *HandoverTokenRepository) ListActiveWithCode(ctx context.Context, transactionID string, now time.Time) ([]*domain.HandoverToken, error) {
	var rows []handoverTokenRow
	err := conn(ctx, r.db).
		Where("transaction_id = ? AND code_hash IS NOT NULL AND consumed_at IS NULL AND expires_at > ?", transactionID, formatTime(now)).
		Order("created_at DESC, id DESC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]*domain.HandoverToken, 0, len(rows))
	for _, row := range rows {
		t, err := row.toDomain()
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

// Consume は未使用・期限内のトークンを使用済みにする（DEAL-3）。
// 条件付きUPDATE なので、同じトークンは一度しか使えない。使えないトークンなら ErrConflict。
func (r *HandoverTokenRepository) Consume(ctx context.Context, id string, now time.Time) error {
	ts := formatTime(now)
	res := conn(ctx, r.db).Model(&handoverTokenRow{}).
		Where("id = ? AND consumed_at IS NULL AND expires_at > ?", id, ts).
		Updates(map[string]any{"consumed_at": ts, "updated_at": ts})
	return rowsAffected(res)
}
