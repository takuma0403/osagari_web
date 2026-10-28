package handler

import (
	"context"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/takuma0403/osagari_web/server/internal/api"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/service"
)

// AdminUserService は管理者によるユーザーの操作（service.AdminUserService）。
type AdminUserService interface {
	List(ctx context.Context, admin *domain.User, q service.AdminUserListQuery) (*service.AdminUserPage, error)
	Get(ctx context.Context, admin *domain.User, id string) (*service.AdminUserView, error)
	Suspend(ctx context.Context, admin *domain.User, id string) (*service.AdminUserView, error)
	Unsuspend(ctx context.Context, admin *domain.User, id string) (*service.AdminUserView, error)
	UpdateExpiresOn(ctx context.Context, admin *domain.User, id string, expiresOn time.Time) (*service.AdminUserView, error)
	Delete(ctx context.Context, admin *domain.User, id string) (*service.AdminUserView, error)
}

// ListAdminUsers はユーザーを登録の新しい順に返す（GET /api/admin/users。ADMIN, ADR 0034）。
func (h *Handler) ListAdminUsers(ctx context.Context, req api.ListAdminUsersRequestObject) (api.ListAdminUsersResponseObject, error) {
	u, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	p := req.Params
	page, err := h.d.AdminUser.List(ctx, u, service.AdminUserListQuery{
		Q:              p.Q,
		Filter:         (*domain.AdminUserFilter)(p.Filter),
		AdminPageQuery: service.AdminPageQuery{Cursor: p.Cursor, Limit: p.Limit},
	})
	if err != nil {
		return nil, err
	}
	res := api.ListAdminUsers200JSONResponse{
		Entries: make([]api.AdminUser, len(page.Entries)), WithdrawalRequestCount: page.WithdrawalRequestCount, NextCursor: page.NextCursor,
	}
	for i, v := range page.Entries {
		res.Entries[i] = toAdminUser(v)
	}
	return res, nil
}

// GetAdminUser はユーザー1人を返す（GET /api/admin/users/{id}。ADMIN）。
func (h *Handler) GetAdminUser(ctx context.Context, req api.GetAdminUserRequestObject) (api.GetAdminUserResponseObject, error) {
	v, err := h.adminUser(ctx, req.UserId, h.d.AdminUser.Get)
	if err != nil {
		return nil, err
	}
	return api.GetAdminUser200JSONResponse(v), nil
}

// SuspendUser はアカウントを停止する（POST /api/admin/users/{id}/suspend。ADMIN, ADR 0034）。
func (h *Handler) SuspendUser(ctx context.Context, req api.SuspendUserRequestObject) (api.SuspendUserResponseObject, error) {
	v, err := h.adminUser(ctx, req.UserId, h.d.AdminUser.Suspend)
	if err != nil {
		return nil, err
	}
	return api.SuspendUser200JSONResponse(v), nil
}

// UnsuspendUser はアカウントの停止を解除する（POST /api/admin/users/{id}/unsuspend。ADMIN, ADR 0034）。
func (h *Handler) UnsuspendUser(ctx context.Context, req api.UnsuspendUserRequestObject) (api.UnsuspendUserResponseObject, error) {
	v, err := h.adminUser(ctx, req.UserId, h.d.AdminUser.Unsuspend)
	if err != nil {
		return nil, err
	}
	return api.UnsuspendUser200JSONResponse(v), nil
}

// UpdateUserExpiresOn は有効期限を変える（PUT /api/admin/users/{id}/expires-on。ADMIN, AUTH-3, ADR 0034）。
func (h *Handler) UpdateUserExpiresOn(ctx context.Context, req api.UpdateUserExpiresOnRequestObject) (api.UpdateUserExpiresOnResponseObject, error) {
	if req.Body == nil {
		return nil, errMissingBody
	}
	v, err := h.adminUser(ctx, req.UserId, func(ctx context.Context, admin *domain.User, id string) (*service.AdminUserView, error) {
		return h.d.AdminUser.UpdateExpiresOn(ctx, admin, id, req.Body.ExpiresOn.Time)
	})
	if err != nil {
		return nil, err
	}
	return api.UpdateUserExpiresOn200JSONResponse(v), nil
}

// DeleteUser は退学によりアカウントを削除する（POST /api/admin/users/{id}/delete。ADMIN, AUTH-3, ADR 0034）。
func (h *Handler) DeleteUser(ctx context.Context, req api.DeleteUserRequestObject) (api.DeleteUserResponseObject, error) {
	v, err := h.adminUser(ctx, req.UserId, h.d.AdminUser.Delete)
	if err != nil {
		return nil, err
	}
	return api.DeleteUser200JSONResponse(v), nil
}

// adminUser はログイン中の管理者で fn を呼び、応答の形にする。
func (h *Handler) adminUser(ctx context.Context, id string, fn func(ctx context.Context, admin *domain.User, id string) (*service.AdminUserView, error)) (api.AdminUser, error) {
	u, err := currentUser(ctx)
	if err != nil {
		return api.AdminUser{}, err
	}
	v, err := fn(ctx, u, id)
	if err != nil {
		return api.AdminUser{}, err
	}
	return toAdminUser(v), nil
}

func toAdminUser(v *service.AdminUserView) api.AdminUser {
	return api.AdminUser{
		Id:                     v.ID,
		StudentNumber:          v.StudentNumber,
		DepartmentName:         v.DepartmentName(),
		AdmissionYear:          v.AdmissionYear,
		ExpiresOn:              openapi_types.Date{Time: v.ExpiresOn},
		Status:                 api.UserStatus(v.Status),
		SellerStatus:           api.SellerStatus(v.SellerStatus),
		CancelCount:            v.CancelCount,
		ActiveTransactionCount: v.ActiveTransactionCount,
		WithdrawalRequestedAt:  v.WithdrawalRequestedAt,
		CreatedAt:              v.CreatedAt,
	}
}
