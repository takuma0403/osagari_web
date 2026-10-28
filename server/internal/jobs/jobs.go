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

// Runner は steps を順に行う定期処理（短周期ジョブ・日次ジョブ。JOBS-1）。
type Runner struct {
	// name はログの見出しに使うジョブの名前
	name   string
	logger *slog.Logger
	steps  []Step
}

// NewShort は steps を順に行う短周期ジョブ（15分ごと）を返す。
func NewShort(logger *slog.Logger, steps ...Step) *Runner {
	return &Runner{name: "短周期ジョブ", logger: logger, steps: steps}
}

// NewDaily は steps を順に行う日次ジョブ（毎日深夜。ADR 0033）を返す。
func NewDaily(logger *slog.Logger, steps ...Step) *Runner {
	return &Runner{name: "日次ジョブ", logger: logger, steps: steps}
}

// Run はすべての処理を順に行い、処理ごとの件数をログに出す。
// 1つが失敗してもほかの処理は続け、失敗をまとめて返す（Cloud Scheduler が再試行する。どの処理も何度行っても結果は同じ）。
func (r *Runner) Run(ctx context.Context) error {
	var errs []error
	for _, st := range r.steps {
		n, err := st.Run(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("jobs: %s: %w", st.Name, err))
			continue
		}
		r.logger.InfoContext(ctx, r.name, "step", st.Name, "count", n)
	}
	return errors.Join(errs...)
}
