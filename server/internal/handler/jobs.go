package handler

import (
	"context"

	"github.com/takuma0403/osagari_web/server/internal/api"
	"github.com/takuma0403/osagari_web/server/internal/apperr"
)

// ShortJob は短周期ジョブ（jobs.Short）。
type ShortJob interface {
	Run(ctx context.Context) error
}

// RunShortJob は短周期ジョブを実行する（POST /api/internal/jobs/short。JOBS-1）。
// 呼び出し元（Cloud Scheduler）の確認は internal/server が行う。
func (h *Handler) RunShortJob(ctx context.Context, _ api.RunShortJobRequestObject) (api.RunShortJobResponseObject, error) {
	if err := h.d.ShortJob.Run(ctx); err != nil {
		return nil, apperr.Internal(err)
	}
	return api.RunShortJob204Response{}, nil
}
