package handler

import (
	"context"

	"github.com/takuma0403/osagari_web/server/internal/api"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/service"
)

// StudentCardService は学生証の業務ロジック（service.StudentCardService）。
type StudentCardService interface {
	CreateUpload(ctx context.Context, u *domain.User, contentType string) (service.StudentCardUpload, error)
	Submit(ctx context.Context, u *domain.User, objectKey string) (service.StudentCardState, error)
	Current(ctx context.Context, u *domain.User) (service.StudentCardState, error)
}

// CreateStudentCardUpload は学生証画像のアップロード用の署名付きURLを発行する（POST /api/student-cards/uploads。CARD, STORAGE）。
func (h *Handler) CreateStudentCardUpload(ctx context.Context, req api.CreateStudentCardUploadRequestObject) (api.CreateStudentCardUploadResponseObject, error) {
	if req.Body == nil {
		return nil, errMissingBody
	}
	u, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	up, err := h.d.StudentCard.CreateUpload(ctx, u, string(req.Body.ContentType))
	if err != nil {
		return nil, err
	}
	return api.CreateStudentCardUpload201JSONResponse{
		ObjectKey: up.ObjectKey,
		UploadUrl: up.URL,
		Method:    api.StudentCardUploadMethod(up.Method),
		Headers:   up.Headers,
		ExpiresAt: up.ExpiresAt,
	}, nil
}

// SubmitStudentCard はアップロードした学生証画像を登録して審査待ちにする（POST /api/student-cards。CARD, STORAGE）。
func (h *Handler) SubmitStudentCard(ctx context.Context, req api.SubmitStudentCardRequestObject) (api.SubmitStudentCardResponseObject, error) {
	if req.Body == nil {
		return nil, errMissingBody
	}
	u, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	st, err := h.d.StudentCard.Submit(ctx, u, req.Body.ObjectKey)
	if err != nil {
		return nil, err
	}
	return api.SubmitStudentCard201JSONResponse(toStudentCardState(st)), nil
}

// GetCurrentStudentCard は本人の学生証の審査状況を返す（GET /api/student-cards/current。CARD）。
func (h *Handler) GetCurrentStudentCard(ctx context.Context, _ api.GetCurrentStudentCardRequestObject) (api.GetCurrentStudentCardResponseObject, error) {
	u, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	st, err := h.d.StudentCard.Current(ctx, u)
	if err != nil {
		return nil, err
	}
	return api.GetCurrentStudentCard200JSONResponse(toStudentCardState(st)), nil
}

func toStudentCardState(st service.StudentCardState) api.StudentCardState {
	return api.StudentCardState{
		Status:      api.SellerStatus(st.Status),
		SubmittedAt: st.SubmittedAt,
		ReviewedAt:  st.ReviewedAt,
	}
}
