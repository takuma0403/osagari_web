// Package handler は生成した StrictServerInterface を実装する。入力の変換と応答だけを書き、業務ロジックは service に置く。
package handler

import (
	"context"

	"github.com/takuma0403/osagari_web/server/internal/api"
)

// Deps は handler が使う service。
type Deps struct {
	Signup        SignupService
	Login         LoginService
	Session       SessionService
	PasswordReset PasswordResetService
	StudentCard   StudentCardService
}

// Handler は api.StrictServerInterface の実装。
type Handler struct {
	d Deps
}

var _ api.StrictServerInterface = (*Handler)(nil)

func New(d Deps) *Handler {
	return &Handler{d: d}
}

// GetHealth は稼働確認（GET /api/health）。
func (h *Handler) GetHealth(ctx context.Context, _ api.GetHealthRequestObject) (api.GetHealthResponseObject, error) {
	return api.GetHealth200JSONResponse{Status: api.Ok}, nil
}
