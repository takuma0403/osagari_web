package handler

import (
	"context"

	"github.com/takuma0403/osagari_web/server/internal/api"
)

// ReceivePaymentWebhook は決済サービスからの通知の受け口（POST /api/webhooks/payment）。
// フェーズ1は決済がモックなので、何もせず 204 を返す（JOBS-2, ADR 0021）。
func (h *Handler) ReceivePaymentWebhook(_ context.Context, _ api.ReceivePaymentWebhookRequestObject) (api.ReceivePaymentWebhookResponseObject, error) {
	return api.ReceivePaymentWebhook204Response{}, nil
}
