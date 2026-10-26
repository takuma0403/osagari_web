// Package jobs は Cloud Scheduler から内部用エンドポイント経由で呼ぶ定期処理をまとめる（JOBS-1）。
package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

// Step は定期処理の1つの処理。Run は処理した件数を返す。
type Step struct {
	// Name はログに出す処理の名前
	Name string
	Run  func(ctx context.Context) (int, error)
}

// Short は短周期ジョブ（15分ごと。JOBS-1）。
type Short struct {
	logger *slog.Logger
	steps  []Step
}

// NewShort は steps を順に行う短周期ジョブを返す。
func NewShort(logger *slog.Logger, steps ...Step) *Short {
	return &Short{logger: logger, steps: steps}
}

// Run はすべての処理を順に行い、処理ごとの件数をログに出す。
// 1つが失敗してもほかの処理は続け、失敗をまとめて返す（Cloud Scheduler が再試行する。どの処理も何度行っても結果は同じ）。
func (s *Short) Run(ctx context.Context) error {
	var errs []error
	for _, st := range s.steps {
		n, err := st.Run(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("jobs: %s: %w", st.Name, err))
			continue
		}
		s.logger.InfoContext(ctx, "短周期ジョブ", "step", st.Name, "count", n)
	}
	return errors.Join(errs...)
}
