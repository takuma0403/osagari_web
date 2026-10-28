package handler

import (
	"context"

	"github.com/takuma0403/osagari_web/server/internal/api"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/service"
)

// AdminDealService は管理者による出品・取引の操作（service.AdminDealService）。
type AdminDealService interface {
	ListItems(ctx context.Context, admin *domain.User, q service.AdminPageQuery) (*service.AdminItemPage, error)
	SuspendItem(ctx context.Context, admin *domain.User, id string) (*service.AdminItemView, error)
	ListTransactions(ctx context.Context, admin *domain.User, q service.AdminPageQuery) (*service.AdminTransactionPage, error)
	CancelTransaction(ctx context.Context, admin *domain.User, id string) (*service.AdminTransactionView, error)
}

// ListAdminItems はすべての出品を作った日時の新しい順に返す（GET /api/admin/items。ADMIN）。
func (h *Handler) ListAdminItems(ctx context.Context, req api.ListAdminItemsRequestObject) (api.ListAdminItemsResponseObject, error) {
	u, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	page, err := h.d.AdminDeal.ListItems(ctx, u, service.AdminPageQuery{Cursor: req.Params.Cursor, Limit: req.Params.Limit})
	if err != nil {
		return nil, err
	}
	res := api.ListAdminItems200JSONResponse{Entries: make([]api.AdminItem, len(page.Entries)), NextCursor: page.NextCursor}
	for i, v := range page.Entries {
		res.Entries[i] = toAdminItem(v)
	}
	return res, nil
}

// SuspendItem は出品を停止する（POST /api/admin/items/{id}/suspend。ADMIN, ITEM-1, ADR 0034）。
func (h *Handler) SuspendItem(ctx context.Context, req api.SuspendItemRequestObject) (api.SuspendItemResponseObject, error) {
	u, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	v, err := h.d.AdminDeal.SuspendItem(ctx, u, req.ItemId)
	if err != nil {
		return nil, err
	}
	return api.SuspendItem200JSONResponse(toAdminItem(v)), nil
}

// ListAdminTransactions はすべての取引を最後に動いた日時の新しい順に返す（GET /api/admin/transactions。ADMIN）。
func (h *Handler) ListAdminTransactions(ctx context.Context, req api.ListAdminTransactionsRequestObject) (api.ListAdminTransactionsResponseObject, error) {
	u, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	page, err := h.d.AdminDeal.ListTransactions(ctx, u, service.AdminPageQuery{Cursor: req.Params.Cursor, Limit: req.Params.Limit})
	if err != nil {
		return nil, err
	}
	res := api.ListAdminTransactions200JSONResponse{Entries: make([]api.AdminTransaction, len(page.Entries)), NextCursor: page.NextCursor}
	for i, v := range page.Entries {
		res.Entries[i] = toAdminTransaction(v)
	}
	return res, nil
}

// CancelTransactionByAdmin は取引をキャンセルして返金する（モック。POST /api/admin/transactions/{id}/cancel。ADMIN, DEAL-4, ADR 0034）。
func (h *Handler) CancelTransactionByAdmin(ctx context.Context, req api.CancelTransactionByAdminRequestObject) (api.CancelTransactionByAdminResponseObject, error) {
	u, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	v, err := h.d.AdminDeal.CancelTransaction(ctx, u, req.TransactionId)
	if err != nil {
		return nil, err
	}
	return api.CancelTransactionByAdmin200JSONResponse(toAdminTransaction(v)), nil
}

func toAdminItem(v *service.AdminItemView) api.AdminItem {
	it := v.Item
	out := api.AdminItem{
		Id:                  it.ID,
		ShareId:             it.ShareID,
		Title:               it.Title,
		Price:               it.Price,
		Status:              api.ItemStatus(it.Status),
		SellerStudentNumber: v.SellerStudentNumber,
		CreatedAt:           it.CreatedAt,
	}
	if images := toItemImages(it); len(images) > 0 {
		out.Thumbnail = &images[0]
	}
	return out
}

func toAdminTransaction(v *service.AdminTransactionView) api.AdminTransaction {
	return api.AdminTransaction{
		Id:                   v.ID,
		Status:               api.TransactionStatus(v.Status),
		Item:                 api.TransactionItem{ShareId: v.Item.ShareID, Title: v.Item.Title, Status: api.ItemStatus(v.Item.Status)},
		SellerStudentNumber:  v.SellerStudentNumber,
		BuyerStudentNumber:   v.BuyerStudentNumber,
		Price:                v.Price,
		PaymentMethod:        api.PaymentMethod(v.PaymentMethod),
		FeeRateBasisPoints:   v.FeeRateBP,
		BuyerCancelAgreedAt:  v.BuyerCancelAgreedAt,
		SellerCancelAgreedAt: v.SellerCancelAgreedAt,
		CompletedAt:          v.CompletedAt,
		CreatedAt:            v.CreatedAt,
		UpdatedAt:            v.UpdatedAt,
	}
}
