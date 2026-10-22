package handler

import (
	"context"

	"github.com/takuma0403/osagari_web/server/internal/api"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/service"
)

// DealService は購入希望・承認・却下の業務ロジック（service.DealService）。
type DealService interface {
	CreateRequest(ctx context.Context, u *domain.User, shareID string, method domain.PaymentMethod) (*service.TransactionView, error)
	ListRequests(ctx context.Context, u *domain.User, itemID string) ([]*service.PurchaseRequestView, error)
	Get(ctx context.Context, u *domain.User, id string) (*service.TransactionView, error)
	Approve(ctx context.Context, u *domain.User, id string) (*service.TransactionView, error)
	Reject(ctx context.Context, u *domain.User, id string) (*service.TransactionView, error)
	Acknowledge(ctx context.Context, u *domain.User, id string) (*service.TransactionView, error)
}

// CreatePurchaseRequest は購入希望を出す（POST /api/items/shared/{shareId}/requests。DEAL-1）。
func (h *Handler) CreatePurchaseRequest(ctx context.Context, req api.CreatePurchaseRequestRequestObject) (api.CreatePurchaseRequestResponseObject, error) {
	if req.Body == nil {
		return nil, errMissingBody
	}
	u, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	v, err := h.d.Deal.CreateRequest(ctx, u, req.ShareId, domain.PaymentMethod(req.Body.PaymentMethod))
	if err != nil {
		return nil, err
	}
	return api.CreatePurchaseRequest201JSONResponse(toTransaction(v)), nil
}

// ListPurchaseRequests は自分の商品に届いている購入希望を返す（GET /api/items/{itemId}/requests。DEAL-1）。
func (h *Handler) ListPurchaseRequests(ctx context.Context, req api.ListPurchaseRequestsRequestObject) (api.ListPurchaseRequestsResponseObject, error) {
	u, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	reqs, err := h.d.Deal.ListRequests(ctx, u, req.ItemId)
	if err != nil {
		return nil, err
	}
	out := api.PurchaseRequestList{Requests: make([]api.PurchaseRequest, len(reqs))}
	for i, r := range reqs {
		out.Requests[i] = api.PurchaseRequest{
			TransactionId:    r.ID,
			RequestedAt:      r.CreatedAt,
			ApprovalDeadline: r.ApprovalDeadline,
			Buyer: api.RequestBuyer{
				DepartmentName:     r.Buyer.DepartmentName,
				AdmissionYear:      r.Buyer.AdmissionYear,
				CompletedDealCount: r.Buyer.CompletedDealCount,
			},
		}
	}
	return api.ListPurchaseRequests200JSONResponse(out), nil
}

// GetTransaction は取引の詳細を返す（GET /api/transactions/{transactionId}。DEAL-1, DEAL-2）。
func (h *Handler) GetTransaction(ctx context.Context, req api.GetTransactionRequestObject) (api.GetTransactionResponseObject, error) {
	v, err := h.transaction(ctx, req.TransactionId, h.d.Deal.Get)
	if err != nil {
		return nil, err
	}
	return api.GetTransaction200JSONResponse(v), nil
}

// ApproveTransaction は購入希望を承認する（POST /api/transactions/{transactionId}/approve。DEAL-1）。
func (h *Handler) ApproveTransaction(ctx context.Context, req api.ApproveTransactionRequestObject) (api.ApproveTransactionResponseObject, error) {
	v, err := h.transaction(ctx, req.TransactionId, h.d.Deal.Approve)
	if err != nil {
		return nil, err
	}
	return api.ApproveTransaction200JSONResponse(v), nil
}

// RejectTransaction は購入希望を却下する（POST /api/transactions/{transactionId}/reject。DEAL-1）。
func (h *Handler) RejectTransaction(ctx context.Context, req api.RejectTransactionRequestObject) (api.RejectTransactionResponseObject, error) {
	v, err := h.transaction(ctx, req.TransactionId, h.d.Deal.Reject)
	if err != nil {
		return nil, err
	}
	return api.RejectTransaction200JSONResponse(v), nil
}

// AcknowledgeTransaction は取引成立の確認モーダルへの同意を記録する（POST /api/transactions/{transactionId}/acknowledge。DEAL-2）。
func (h *Handler) AcknowledgeTransaction(ctx context.Context, req api.AcknowledgeTransactionRequestObject) (api.AcknowledgeTransactionResponseObject, error) {
	v, err := h.transaction(ctx, req.TransactionId, h.d.Deal.Acknowledge)
	if err != nil {
		return nil, err
	}
	return api.AcknowledgeTransaction200JSONResponse(v), nil
}

// transaction はログイン中の本人で取引の操作 op を呼び、応答の形にする。
func (h *Handler) transaction(ctx context.Context, id string, op func(context.Context, *domain.User, string) (*service.TransactionView, error)) (api.Transaction, error) {
	u, err := currentUser(ctx)
	if err != nil {
		return api.Transaction{}, err
	}
	v, err := op(ctx, u, id)
	if err != nil {
		return api.Transaction{}, err
	}
	return toTransaction(v), nil
}

func toTransaction(v *service.TransactionView) api.Transaction {
	return api.Transaction{
		Id:                       v.ID,
		Role:                     api.TransactionRole(v.Role),
		Status:                   api.TransactionStatus(v.Status),
		Item:                     api.TransactionItem{ShareId: v.Item.ShareID, Title: v.Item.Title, Status: api.ItemStatus(v.Item.Status)},
		Price:                    v.Price,
		PaymentMethod:            api.PaymentMethod(v.PaymentMethod),
		ApprovalDeadline:         v.ApprovalDeadline,
		FeeRateBasisPoints:       v.FeeRateBP,
		Fee:                      v.Fee,
		SellerProceeds:           v.SellerProceeds,
		IdentityDisclosedAt:      v.IdentityDisclosedAt,
		SellerAcknowledgedAt:     v.SellerAcknowledgedAt,
		CounterpartStudentNumber: v.CounterpartStudentNumber,
		CreatedAt:                v.CreatedAt,
	}
}
