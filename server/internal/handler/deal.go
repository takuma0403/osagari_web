package handler

import (
	"context"

	"github.com/takuma0403/osagari_web/server/internal/api"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/service"
)

// DealService は購入希望・承認・却下・受け渡し完了・キャンセルの業務ロジック（service.DealService）。
type DealService interface {
	CreateRequest(ctx context.Context, u *domain.User, shareID string, method domain.PaymentMethod) (*service.TransactionView, error)
	ListRequests(ctx context.Context, u *domain.User, itemID string) ([]*service.PurchaseRequestView, error)
	Get(ctx context.Context, u *domain.User, id string) (*service.TransactionView, error)
	Approve(ctx context.Context, u *domain.User, id string) (*service.TransactionView, error)
	Reject(ctx context.Context, u *domain.User, id string) (*service.TransactionView, error)
	Acknowledge(ctx context.Context, u *domain.User, id string) (*service.TransactionView, error)
	IssueHandoverToken(ctx context.Context, u *domain.User, id string, kind domain.HandoverTokenKind) (*service.IssuedHandoverToken, error)
	VerifyHandover(ctx context.Context, u *domain.User, id string, cred service.HandoverCredential) error
	Complete(ctx context.Context, u *domain.User, id string, cred service.HandoverCredential) (*service.TransactionView, error)
	Cancel(ctx context.Context, u *domain.User, id string) (*service.TransactionView, error)
	WithdrawCancel(ctx context.Context, u *domain.User, id string) (*service.TransactionView, error)
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

// CancelTransaction は取引のキャンセルを申請する・申請に同意する（POST /api/transactions/{transactionId}/cancel。DEAL-4）。
func (h *Handler) CancelTransaction(ctx context.Context, req api.CancelTransactionRequestObject) (api.CancelTransactionResponseObject, error) {
	v, err := h.transaction(ctx, req.TransactionId, h.d.Deal.Cancel)
	if err != nil {
		return nil, err
	}
	return api.CancelTransaction200JSONResponse(v), nil
}

// WithdrawCancelRequest はキャンセルの申請を取り下げる（POST /api/transactions/{transactionId}/cancel/withdraw。DEAL-4）。
func (h *Handler) WithdrawCancelRequest(ctx context.Context, req api.WithdrawCancelRequestRequestObject) (api.WithdrawCancelRequestResponseObject, error) {
	v, err := h.transaction(ctx, req.TransactionId, h.d.Deal.WithdrawCancel)
	if err != nil {
		return nil, err
	}
	return api.WithdrawCancelRequest200JSONResponse(v), nil
}

// CreateHandoverToken は受け渡しトークンを発行する（POST /api/transactions/{transactionId}/handover-tokens。DEAL-3）。
func (h *Handler) CreateHandoverToken(ctx context.Context, req api.CreateHandoverTokenRequestObject) (api.CreateHandoverTokenResponseObject, error) {
	if req.Body == nil {
		return nil, errMissingBody
	}
	u, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	t, err := h.d.Deal.IssueHandoverToken(ctx, u, req.TransactionId, domain.HandoverTokenKind(req.Body.Kind))
	if err != nil {
		return nil, err
	}
	return api.CreateHandoverToken201JSONResponse{
		Kind:      api.HandoverTokenKind(t.Kind),
		Token:     t.Token,
		Code:      t.Code,
		ExpiresAt: t.ExpiresAt,
	}, nil
}

// VerifyHandover は受け渡しトークンを照合する（POST /api/transactions/{transactionId}/handover/verify。DEAL-3）。
func (h *Handler) VerifyHandover(ctx context.Context, req api.VerifyHandoverRequestObject) (api.VerifyHandoverResponseObject, error) {
	if req.Body == nil {
		return nil, errMissingBody
	}
	u, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.d.Deal.VerifyHandover(ctx, u, req.TransactionId, toHandoverCredential(*req.Body)); err != nil {
		return nil, err
	}
	return api.VerifyHandover204Response{}, nil
}

// CompleteTransaction は受け渡しを完了する（POST /api/transactions/{transactionId}/complete。DEAL-3）。
func (h *Handler) CompleteTransaction(ctx context.Context, req api.CompleteTransactionRequestObject) (api.CompleteTransactionResponseObject, error) {
	if req.Body == nil {
		return nil, errMissingBody
	}
	cred := toHandoverCredential(*req.Body)
	v, err := h.transaction(ctx, req.TransactionId, func(ctx context.Context, u *domain.User, id string) (*service.TransactionView, error) {
		return h.d.Deal.Complete(ctx, u, id, cred)
	})
	if err != nil {
		return nil, err
	}
	return api.CompleteTransaction200JSONResponse(v), nil
}

func toHandoverCredential(b api.HandoverCredential) service.HandoverCredential {
	return service.HandoverCredential{Kind: domain.HandoverTokenKind(b.Kind), Secret: b.Secret}
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
		BuyerCancelAgreedAt:      v.BuyerCancelAgreedAt,
		SellerCancelAgreedAt:     v.SellerCancelAgreedAt,
		CompletedAt:              v.CompletedAt,
		HandoverMethod:           (*api.HandoverMethod)(v.HandoverMethod),
		CounterpartStudentNumber: v.CounterpartStudentNumber,
		CreatedAt:                v.CreatedAt,
	}
}
