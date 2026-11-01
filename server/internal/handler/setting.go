package handler

import (
	"context"

	"github.com/takuma0403/osagari_web/server/internal/api"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/service"
)

// SettingService は設定の業務ロジック（service.SettingService）。
type SettingService interface {
	FeeRate(ctx context.Context) (int, error)
	AdminFeeRate(ctx context.Context, admin *domain.User) (*service.AdminFeeRateView, error)
	UpdateFeeRate(ctx context.Context, admin *domain.User, bp int) (*service.AdminFeeRateView, error)
}

// GetFeeRate は現在の手数料率を返す（GET /api/settings/fee-rate。PAY-2）。
func (h *Handler) GetFeeRate(ctx context.Context, _ api.GetFeeRateRequestObject) (api.GetFeeRateResponseObject, error) {
	bp, err := h.d.Setting.FeeRate(ctx)
	if err != nil {
		return nil, err
	}
	return api.GetFeeRate200JSONResponse{FeeRateBasisPoints: bp}, nil
}

// GetAdminFeeRate は手数料率と最終更新を返す（GET /api/admin/settings/fee-rate。ADMIN, PAY-2, ADR 0034）。
func (h *Handler) GetAdminFeeRate(ctx context.Context, _ api.GetAdminFeeRateRequestObject) (api.GetAdminFeeRateResponseObject, error) {
	u, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	v, err := h.d.Setting.AdminFeeRate(ctx, u)
	if err != nil {
		return nil, err
	}
	return api.GetAdminFeeRate200JSONResponse(toAdminFeeRate(v)), nil
}

// UpdateFeeRate は手数料率を変える（PUT /api/admin/settings/fee-rate。ADMIN, PAY-2, ADR 0034）。
func (h *Handler) UpdateFeeRate(ctx context.Context, req api.UpdateFeeRateRequestObject) (api.UpdateFeeRateResponseObject, error) {
	if req.Body == nil {
		return nil, errMissingBody
	}
	u, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	v, err := h.d.Setting.UpdateFeeRate(ctx, u, req.Body.FeeRateBasisPoints)
	if err != nil {
		return nil, err
	}
	return api.UpdateFeeRate200JSONResponse(toAdminFeeRate(v)), nil
}

func toAdminFeeRate(v *service.AdminFeeRateView) api.AdminFeeRate {
	return api.AdminFeeRate{FeeRateBasisPoints: v.BasisPoints, UpdatedAt: v.UpdatedAt, UpdatedByStudentNumber: v.UpdatedByStudentNumber}
}
