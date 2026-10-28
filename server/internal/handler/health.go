// Package handler は生成した StrictServerInterface を実装する。入力の変換と応答だけを書き、業務ロジックは service に置く。
package handler

import (
	"context"

	"github.com/takuma0403/osagari_web/server/internal/api"
)

// Deps は handler が使う service。
type Deps struct {
	Signup        SignupService
	Login         LoginService
	Session       SessionService
	PasswordReset PasswordResetService
	StudentCard   StudentCardService
	// Item は出品・編集・取り下げ（ITEM）
	Item ItemService
	// Deal は購入希望・承認・却下と取引の詳細（DEAL-1, DEAL-2）
	Deal DealService
	// Sales は売上管理と出金申請（PAY, ADR 0029）
	Sales SalesService
	// Setting は手数料率などの設定（PAY-2）
	Setting SettingService
	// AdminStudentCard は管理者による学生証の審査（ADMIN）
	AdminStudentCard AdminStudentCardService
	// AdminUser は管理者によるユーザーの検索・停止・有効期限の変更・削除（ADMIN, ADR 0034）
	AdminUser AdminUserService
	// AdminDeal は管理者による出品・取引の確認、出品の停止、取引のキャンセル（ADMIN, ADR 0034）
	AdminDeal AdminDealService
	// ShortJob は15分ごとの短周期ジョブ（JOBS-1）
	ShortJob Job
	// DailyJob は毎日深夜の日次ジョブ（JOBS-1, ADR 0033）
	DailyJob Job
}

// Handler は api.StrictServerInterface の実装。
type Handler struct {
	d Deps
}

var _ api.StrictServerInterface = (*Handler)(nil)

func New(d Deps) *Handler {
	return &Handler{d: d}
}

// GetHealth は稼働確認（GET /api/health）。
func (h *Handler) GetHealth(ctx context.Context, _ api.GetHealthRequestObject) (api.GetHealthResponseObject, error) {
	return api.GetHealth200JSONResponse{Status: api.Ok}, nil
}
