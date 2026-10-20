package service

import (
	"context"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
)

// SettingDeps は SettingService の依存。
type SettingDeps struct {
	Settings SettingStore
}

// SettingService は手数料率などの設定を扱う（PAY-2）。変更は管理画面（ADMIN）で行う。
type SettingService struct {
	d SettingDeps
}

func NewSettingService(d SettingDeps) *SettingService {
	return &SettingService{d: d}
}

// FeeRate は現在の手数料率（ベーシスポイント）を返す。出品画面の受け取り見込みの表示に使う。
func (s *SettingService) FeeRate(ctx context.Context) (int, error) {
	bp, err := s.d.Settings.FeeRate(ctx)
	if err != nil {
		return 0, apperr.Internal(err)
	}
	return bp, nil
}
