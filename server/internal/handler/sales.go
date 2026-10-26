package handler

import (
	"context"

	"github.com/takuma0403/osagari_web/server/internal/api"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/service"
)

// SalesService は売上管理と出金申請の業務ロジック（service.SalesService）。
type SalesService interface {
	Get(ctx context.Context, u *domain.User) (*service.SalesView, error)
	RequestPayout(ctx context.Context, u *domain.User) (*service.SalesView, error)
}

// GetSales は本人の売上の明細と集計を返す（GET /api/me/sales。PAY, PAY-2）。
func (h *Handler) GetSales(ctx context.Context, _ api.GetSalesRequestObject) (api.GetSalesResponseObject, error) {
	v, err := h.sales(ctx, h.d.Sales.Get)
	if err != nil {
		return nil, err
	}
	return api.GetSales200JSONResponse(v), nil
}

// RequestPayout は出金できる売上をすべて出金する（POST /api/me/sales/payouts。PAY, JOBS-2。モック）。
func (h *Handler) RequestPayout(ctx context.Context, _ api.RequestPayoutRequestObject) (api.RequestPayoutResponseObject, error) {
	v, err := h.sales(ctx, h.d.Sales.RequestPayout)
	if err != nil {
		return nil, err
	}
	return api.RequestPayout200JSONResponse(v), nil
}

func (h *Handler) sales(ctx context.Context, op func(context.Context, *domain.User) (*service.SalesView, error)) (api.Sales, error) {
	u, err := currentUser(ctx)
	if err != nil {
		return api.Sales{}, err
	}
	v, err := op(ctx, u)
	if err != nil {
		return api.Sales{}, err
	}
	return toSales(v), nil
}

// toSales は売上の集計と明細にする。金額は記録済みの値をそのまま出す（PAY-2）。
func toSales(v *service.SalesView) api.Sales {
	out := api.Sales{
		AvailableAmount:        v.Available,
		AwaitingHandoverAmount: v.AwaitingHandover,
		PaidOutAmount:          v.PaidOut,
		Entries:                make([]api.SalesEntry, len(v.Entries)),
	}
	for i, e := range v.Entries {
		tr := e.Transaction
		out.Entries[i] = api.SalesEntry{
			TransactionId:      tr.ID,
			Item:               toTransactionListItem(e.Item),
			State:              api.SalesEntryState(e.State()),
			Price:              tr.Price,
			FeeRateBasisPoints: deref(tr.FeeRateBP),
			Fee:                deref(tr.Fee),
			SellerProceeds:     deref(tr.SellerProceeds),
			ApprovedAt:         deref(tr.IdentityDisclosedAt),
			CompletedAt:        tr.CompletedAt,
			PaidOutAt:          e.PaidOutAt,
		}
	}
	return out
}
