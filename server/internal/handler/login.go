package handler

import (
	"context"

	"github.com/takuma0403/osagari_web/server/internal/api"
	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/auth"
	"github.com/takuma0403/osagari_web/server/internal/service"
)

// LoginService はログインと再確認の業務ロジック（service.LoginService）。
type LoginService interface {
	Login(ctx context.Context, email, password, deviceID string) (service.LoginResult, error)
	ResendRecheckCode(ctx context.Context, challengeToken string) error
	VerifyRecheck(ctx context.Context, challengeToken, code string) (service.LoggedIn, error)
}

// SessionService はセッションの業務ロジック（service.SessionService）。
type SessionService interface {
	Logout(ctx context.Context, token string) error
}

// Login はメールアドレスとパスワードでログインする（POST /api/auth/login。AUTHN-3）。
// 再確認が要らなければ 200 と `__session` Cookie、要るなら 202 とログインチャレンジを返す。
func (h *Handler) Login(ctx context.Context, req api.LoginRequestObject) (api.LoginResponseObject, error) {
	if req.Body == nil {
		return nil, errMissingBody
	}
	res, err := h.d.Login.Login(ctx, req.Body.Email, req.Body.Password, deviceIDOf(req.Params.XDeviceId))
	if err != nil {
		return nil, err
	}
	if res.Challenge != nil {
		return api.Login202JSONResponse{
			ChallengeToken: res.Challenge.Token,
			Reason:         api.LoginRecheckReason(res.Challenge.Reason),
			Email:          res.Challenge.Email,
		}, nil
	}
	return api.Login200JSONResponse{
		Body:    toCurrentUser(res.User),
		Headers: api.Login200ResponseHeaders{SetCookie: sessionCookie(*res.Session)},
	}, nil
}

// ResendLoginRecheckCode は再確認のコードを送り直す（POST /api/auth/login/recheck/codes。AUTHN-3）。
func (h *Handler) ResendLoginRecheckCode(ctx context.Context, req api.ResendLoginRecheckCodeRequestObject) (api.ResendLoginRecheckCodeResponseObject, error) {
	if req.Body == nil {
		return nil, errMissingBody
	}
	if err := h.d.Login.ResendRecheckCode(ctx, req.Body.ChallengeToken); err != nil {
		return nil, err
	}
	return api.ResendLoginRecheckCode202Response{}, nil
}

// VerifyLoginRecheck は再確認のコードを確認して `__session` Cookie を返す（POST /api/auth/login/recheck/verify。AUTHN-3）。
func (h *Handler) VerifyLoginRecheck(ctx context.Context, req api.VerifyLoginRecheckRequestObject) (api.VerifyLoginRecheckResponseObject, error) {
	if req.Body == nil {
		return nil, errMissingBody
	}
	out, err := h.d.Login.VerifyRecheck(ctx, req.Body.ChallengeToken, req.Body.Code)
	if err != nil {
		return nil, err
	}
	return api.VerifyLoginRecheck200JSONResponse{
		Body:    toCurrentUser(out.User),
		Headers: api.VerifyLoginRecheck200ResponseHeaders{SetCookie: sessionCookie(out.Session)},
	}, nil
}

// Logout はいまのセッションを無効にして `__session` Cookie を消す（POST /api/auth/logout。AUTHN-5）。
func (h *Handler) Logout(ctx context.Context, _ api.LogoutRequestObject) (api.LogoutResponseObject, error) {
	if err := h.d.Session.Logout(ctx, auth.SessionTokenFrom(ctx)); err != nil {
		return nil, err
	}
	cookie := auth.ClearSessionCookie().String()
	return api.Logout204Response{Headers: api.Logout204ResponseHeaders{SetCookie: &cookie}}, nil
}

// GetMe はログイン中の本人の情報を返す（GET /api/me。AUTH-1, AUTHN-5）。セッションの検証は internal/server のミドルウェアが行う。
func (h *Handler) GetMe(ctx context.Context, _ api.GetMeRequestObject) (api.GetMeResponseObject, error) {
	u, ok := auth.UserFrom(ctx)
	if !ok {
		return nil, apperr.Unauthorized("ログインしてください")
	}
	return api.GetMe200JSONResponse(toCurrentUser(u)), nil
}

// sessionCookie は発行したセッションの `__session` Cookie を Set-Cookie の値にする。
func sessionCookie(s service.IssuedSession) *string {
	cookie := auth.NewSessionCookie(s.Token, s.IssuedAt, s.ExpiresAt).String()
	return &cookie
}

// deviceIDOf は X-Device-Id ヘッダーの値を返す。なければ空。
func deviceIDOf(v *api.DeviceId) string {
	if v == nil {
		return ""
	}
	return *v
}
