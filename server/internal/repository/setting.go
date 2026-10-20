package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// settingRow は settings の1行（migrations/000012_create_settings）。行は id = 1 の1行だけ。
type settingRow struct {
	ID        int
	FeeRateBp int
	CreatedAt string
	UpdatedAt string
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
	var row settingRow
	err := conn(ctx, r.db).Where("id = ?", 1).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}
	return row.FeeRateBp, nil
}
