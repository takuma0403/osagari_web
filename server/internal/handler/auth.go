package handler

import (
	"context"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/takuma0403/osagari_web/server/internal/api"
	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/auth"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/service"
)

// SignupService は新規登録の業務ロジック（service.SignupService）。
type SignupService interface {
	Estimate(email string) (domain.StudentEstimate, error)
	SendCode(ctx context.Context, email string) error
	VerifyCode(ctx context.Context, email, code string) (service.VerifiedSignup, error)
	Complete(ctx context.Context, setupToken, password, deviceID string) (service.CompletedSignup, error)
}

var errMissingBody = apperr.BadRequest("")

// EstimateStudent は学籍番号から学部・入学年度・有効期限を推定する（POST /api/auth/signup/estimate。AUTH-2〜4）。
func (h *Handler) EstimateStudent(ctx context.Context, req api.EstimateStudentRequestObject) (api.EstimateStudentResponseObject, error) {
	if req.Body == nil {
		return nil, errMissingBody
	}
	est, err := h.d.Signup.Estimate(req.Body.Email)
	if err != nil {
		return nil, err
	}
	return api.EstimateStudent200JSONResponse(toStudentEstimate(est)), nil
}

// SendSignupCode は新規登録の認証コードを送る（POST /api/auth/signup/codes。AUTHN-2）。登録済みでも同じ 202 を返す。
func (h *Handler) SendSignupCode(ctx context.Context, req api.SendSignupCodeRequestObject) (api.SendSignupCodeResponseObject, error) {
	if req.Body == nil {
		return nil, errMissingBody
	}
	if err := h.d.Signup.SendCode(ctx, req.Body.Email); err != nil {
		return nil, err
	}
	return api.SendSignupCode202Response{}, nil
}

// VerifySignupCode は新規登録の認証コードを確認する（POST /api/auth/signup/codes/verify。AUTHN-2）。
func (h *Handler) VerifySignupCode(ctx context.Context, req api.VerifySignupCodeRequestObject) (api.VerifySignupCodeResponseObject, error) {
	if req.Body == nil {
		return nil, errMissingBody
	}
	v, err := h.d.Signup.VerifyCode(ctx, req.Body.Email, req.Body.Code)
	if err != nil {
		return nil, err
	}
	return api.VerifySignupCode200JSONResponse{SetupToken: v.SetupToken, Estimate: toStudentEstimate(v.Estimate)}, nil
}

// CompleteSignup はパスワードを設定してアカウントを作り、`__session` Cookie を返す（POST /api/auth/signup/complete。AUTHN-2, AUTHN-5）。
func (h *Handler) CompleteSignup(ctx context.Context, req api.CompleteSignupRequestObject) (api.CompleteSignupResponseObject, error) {
	if req.Body == nil {
		return nil, errMissingBody
	}
	deviceID := ""
	if req.Params.XDeviceId != nil {
		deviceID = *req.Params.XDeviceId
	}
	out, err := h.d.Signup.Complete(ctx, req.Body.SetupToken, req.Body.Password, deviceID)
	if err != nil {
		return nil, err
	}
	cookie := auth.NewSessionCookie(out.Session.Token, out.Session.IssuedAt, out.Session.ExpiresAt).String()
	return api.CompleteSignup201JSONResponse{
		Body:    toCurrentUser(out.User),
		Headers: api.CompleteSignup201ResponseHeaders{SetCookie: &cookie},
	}, nil
}

func toStudentEstimate(est domain.StudentEstimate) api.StudentEstimate {
	out := api.StudentEstimate{
		StudentNumber:  est.StudentNumber,
		AdmissionYear:  est.AdmissionYear,
		DepartmentCode: est.DepartmentCode,
		ExpiresOn:      openapi_types.Date{Time: est.ExpiresOn},
	}
	if est.Department != nil {
		out.Department = &est.Department.Code
		out.DepartmentName = &est.Department.Name
	}
	return out
}

func toCurrentUser(u *domain.User) api.CurrentUser {
	return api.CurrentUser{
		Id:             u.ID,
		Email:          u.Email,
		StudentNumber:  u.StudentNumber,
		AdmissionYear:  u.AdmissionYear,
		DepartmentCode: u.DepartmentCode,
		Department:     u.Department,
		DepartmentName: u.DepartmentName(),
		ExpiresOn:      openapi_types.Date{Time: u.ExpiresOn},
		Status:         api.UserStatus(u.Status),
		SellerStatus:   api.SellerStatus(u.SellerStatus),
	}
}
