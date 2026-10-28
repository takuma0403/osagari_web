package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/takuma0403/osagari_web/server/internal/domain"
)

// settingRow は settings の1行（migrations/000012_create_settings、000018_add_settings_fee_rate_updated_by）。行は id = 1 の1行だけ。
type settingRow struct {
	ID        int
	FeeRateBp int
	// FeeRateUpdatedBy は手数料率を最後に変更した管理者（ADR 0034）。初期値のままなら NULL
	FeeRateUpdatedBy *string
	CreatedAt        string
	UpdatedAt        string
}

func (settingRow) TableName() string { return "settings" }

// SettingRepository は設定（settings）を読み書きする（PAY-2）。
type SettingRepository struct {
	db *gorm.DB
}

func NewSettingRepository(db *gorm.DB) *SettingRepository {
	return &SettingRepository{db: db}
}

// FeeRate は現在の手数料率（ベーシスポイント）を返す。行がなければ ErrNotFound（マイグレーションで必ず入れる）。
func (r *SettingRepository) FeeRate(ctx context.Context) (int, error) {
	row, err := r.find(ctx)
	if err != nil {
		return 0, err
	}
	return row.FeeRateBp, nil
}

// FeeRateSetting は現在の手数料率と、最後に変更した日時・管理者を返す（ADMIN, ADR 0034）。行がなければ ErrNotFound。
func (r *SettingRepository) FeeRateSetting(ctx context.Context) (domain.FeeRateSetting, error) {
	row, err := r.find(ctx)
	if err != nil {
		return domain.FeeRateSetting{}, err
	}
	at, err := parseTime(row.UpdatedAt)
	if err != nil {
		return domain.FeeRateSetting{}, fmt.Errorf("repository: settings.updated_at: %w", err)
	}
	return domain.FeeRateSetting{BasisPoints: row.FeeRateBp, UpdatedAt: at, UpdatedBy: row.FeeRateUpdatedBy}, nil
}

// UpdateFeeRate は手数料率を変え、変更した管理者と日時を記録する（PAY-2, ADR 0034）。行がなければ ErrNotFound。
// 成立済みの取引は取引ごとに料率を持つので変わらない（ADR 0003）。
func (r *SettingRepository) UpdateFeeRate(ctx context.Context, bp int, adminUserID string, now time.Time) error {
	res := conn(ctx, r.db).Model(&settingRow{}).Where("id = ?", 1).
		Updates(map[string]any{"fee_rate_bp": bp, "fee_rate_updated_by": adminUserID, "updated_at": formatTime(now)})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *SettingRepository) find(ctx context.Context) (settingRow, error) {
	var row settingRow
	err := conn(ctx, r.db).Where("id = ?", 1).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return settingRow{}, ErrNotFound
	}
	return row, err
}
