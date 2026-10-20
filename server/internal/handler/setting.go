package handler

import (
	"context"

	"github.com/takuma0403/osagari_web/server/internal/api"
)

// SettingService は設定の業務ロジック（service.SettingService）。
type SettingService interface {
	FeeRate(ctx context.Context) (int, error)
}

// GetFeeRate は現在の手数料率を返す（GET /api/settings/fee-rate。PAY-2）。
func (h *Handler) GetFeeRate(ctx context.Context, _ api.GetFeeRateRequestObject) (api.GetFeeRateResponseObject, error) {
	bp, err := h.d.Setting.FeeRate(ctx)
	if err != nil {
		return nil, err
	}
	return api.GetFeeRate200JSONResponse{FeeRateBasisPoints: bp}, nil
}
