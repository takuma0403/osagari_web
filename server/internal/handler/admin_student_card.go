package handler

import (
	"context"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/takuma0403/osagari_web/server/internal/api"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/service"
)

// AdminStudentCardService は管理者による学生証の審査（service.AdminStudentCardService）。
type AdminStudentCardService interface {
	RequireAdmin(u *domain.User) error
	ListPending(ctx context.Context, admin *domain.User) ([]service.AdminStudentCard, error)
	CreateView(ctx context.Context, admin *domain.User, cardID string) (service.StudentCardImage, error)
	ListViews(ctx context.Context, admin *domain.User, cardID string) ([]service.StudentCardViewEntry, error)
	Approve(ctx context.Context, admin *domain.User, cardID string) (service.AdminStudentCard, error)
	Reject(ctx context.Context, admin *domain.User, cardID string) (service.AdminStudentCard, error)
}

// GetAdminSession は管理者なら 204 を返す（GET /api/admin/session。ADMIN, FE）。
// 管理者でなければミドルウェアが 404 にするが、service でも同じ判定をする（ADR 0015）。
func (h *Handler) GetAdminSession(ctx context.Context, _ api.GetAdminSessionRequestObject) (api.GetAdminSessionResponseObject, error) {
	u, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.d.AdminStudentCard.RequireAdmin(u); err != nil {
		return nil, err
	}
	return api.GetAdminSession204Response{}, nil
}

// ListPendingStudentCards は審査待ちの学生証を古い順に返す（GET /api/admin/student-cards。ADMIN, CARD）。
func (h *Handler) ListPendingStudentCards(ctx context.Context, _ api.ListPendingStudentCardsRequestObject) (api.ListPendingStudentCardsResponseObject, error) {
	u, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	list, err := h.d.AdminStudentCard.ListPending(ctx, u)
	if err != nil {
		return nil, err
	}
	res := make(api.ListPendingStudentCards200JSONResponse, len(list))
	for i, c := range list {
		res[i] = toAdminStudentCard(c)
	}
	return res, nil
}

// CreateStudentCardView は学生証画像の閲覧URLを発行し、閲覧を記録する（POST /api/admin/student-cards/{id}/views。ADMIN, STORAGE）。
func (h *Handler) CreateStudentCardView(ctx context.Context, req api.CreateStudentCardViewRequestObject) (api.CreateStudentCardViewResponseObject, error) {
	u, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	img, err := h.d.AdminStudentCard.CreateView(ctx, u, req.StudentCardId)
	if err != nil {
		return nil, err
	}
	return api.CreateStudentCardView201JSONResponse{Url: img.URL, ExpiresAt: img.ExpiresAt, ViewedAt: img.ViewedAt}, nil
}

// ListStudentCardViews は学生証画像の閲覧記録を新しい順に返す（GET /api/admin/student-cards/{id}/views。ADMIN）。
func (h *Handler) ListStudentCardViews(ctx context.Context, req api.ListStudentCardViewsRequestObject) (api.ListStudentCardViewsResponseObject, error) {
	u, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	views, err := h.d.AdminStudentCard.ListViews(ctx, u, req.StudentCardId)
	if err != nil {
		return nil, err
	}
	res := make(api.ListStudentCardViews200JSONResponse, len(views))
	for i, v := range views {
		res[i] = api.StudentCardView{ViewedAt: v.ViewedAt, AdminStudentNumber: v.AdminStudentNumber}
	}
	return res, nil
}

// ApproveStudentCard は学生証を承認する（POST /api/admin/student-cards/{id}/approve。ADMIN, CARD）。
func (h *Handler) ApproveStudentCard(ctx context.Context, req api.ApproveStudentCardRequestObject) (api.ApproveStudentCardResponseObject, error) {
	u, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	c, err := h.d.AdminStudentCard.Approve(ctx, u, req.StudentCardId)
	if err != nil {
		return nil, err
	}
	return api.ApproveStudentCard200JSONResponse(toAdminStudentCard(c)), nil
}

// RejectStudentCard は学生証を差し戻す（POST /api/admin/student-cards/{id}/reject。ADMIN, CARD）。
func (h *Handler) RejectStudentCard(ctx context.Context, req api.RejectStudentCardRequestObject) (api.RejectStudentCardResponseObject, error) {
	u, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	c, err := h.d.AdminStudentCard.Reject(ctx, u, req.StudentCardId)
	if err != nil {
		return nil, err
	}
	return api.RejectStudentCard200JSONResponse(toAdminStudentCard(c)), nil
}

func toAdminStudentCard(c service.AdminStudentCard) api.AdminStudentCard {
	return api.AdminStudentCard{
		Id:             c.Card.ID,
		Status:         api.StudentCardStatus(c.Card.Status),
		StudentNumber:  c.User.StudentNumber,
		DepartmentName: c.User.DepartmentName(),
		AdmissionYear:  c.User.AdmissionYear,
		ExpiresOn:      openapi_types.Date{Time: c.User.ExpiresOn},
		SubmittedAt:    c.Card.SubmittedAt,
		ReviewedAt:     c.Card.ReviewedAt,
	}
}
