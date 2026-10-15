package handler

import (
	"context"

	"github.com/takuma0403/osagari_web/server/internal/api"
	"github.com/takuma0403/osagari_web/server/internal/service"
)

// PasswordResetService はパスワード再設定の業務ロジック（service.PasswordResetService）。
type PasswordResetService interface {
	SendCode(ctx context.Context, email string) error
	VerifyCode(ctx context.Context, email, code string) (string, error)
	Complete(ctx context.Context, setupToken, password, deviceID string) (service.LoggedIn, error)
}

// SendPasswordResetCode はパスワード再設定の認証コードを送る（POST /api/auth/password-reset/codes。AUTHN-4）。登録の有無にかかわらず同じ 202 を返す。
func (h *Handler) SendPasswordResetCode(ctx context.Context, req api.SendPasswordResetCodeRequestObject) (api.SendPasswordResetCodeResponseObject, error) {
	if req.Body == nil {
		return nil, errMissingBody
	}
	if err := h.d.PasswordReset.SendCode(ctx, req.Body.Email); err != nil {
		return nil, err
	}
	return api.SendPasswordResetCode202Response{}, nil
}

// VerifyPasswordResetCode はパスワード再設定の認証コードを確認する（POST /api/auth/password-reset/codes/verify。AUTHN-4）。
func (h *Handler) VerifyPasswordResetCode(ctx context.Context, req api.VerifyPasswordResetCodeRequestObject) (api.VerifyPasswordResetCodeResponseObject, error) {
	if req.Body == nil {
		return nil, errMissingBody
	}
	token, err := h.d.PasswordReset.VerifyCode(ctx, req.Body.Email, req.Body.Code)
	if err != nil {
		return nil, err
	}
	return api.VerifyPasswordResetCode200JSONResponse{SetupToken: token}, nil
}

// CompletePasswordReset は新しいパスワードを設定し、ほかのセッションを無効にして `__session` Cookie を返す
// （POST /api/auth/password-reset/complete。AUTHN-4, AUTHN-5）。
func (h *Handler) CompletePasswordReset(ctx context.Context, req api.CompletePasswordResetRequestObject) (api.CompletePasswordResetResponseObject, error) {
	if req.Body == nil {
		return nil, errMissingBody
	}
	out, err := h.d.PasswordReset.Complete(ctx, req.Body.SetupToken, req.Body.Password, deviceIDOf(req.Params.XDeviceId))
	if err != nil {
		return nil, err
	}
	return api.CompletePasswordReset200JSONResponse{
		Body:    toCurrentUser(out.User),
		Headers: api.CompletePasswordReset200ResponseHeaders{SetCookie: sessionCookie(out.Session)},
	}, nil
}
