package service

import (
	"context"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/auth"
	"github.com/takuma0403/osagari_web/server/internal/clock"
)

// StaleLoginFailureStore は古い入力ミスの記録を消す（repository.LoginFailureRepository）。
type StaleLoginFailureStore interface {
	DeleteStale(ctx context.Context, before, now time.Time) (int, error)
}

// AuthCleanupDeps は AuthCleanupService の依存。
type AuthCleanupDeps struct {
	Failures StaleLoginFailureStore
	Clock    clock.Clock
}

// AuthCleanupService は認証まわりの古い記録を消す。短周期ジョブ（JOBS-1, #31）から呼ぶ。
type AuthCleanupService struct {
	d AuthCleanupDeps
}

func NewAuthCleanupService(d AuthCleanupDeps) *AuthCleanupService {
	return &AuthCleanupService{d: d}
}

// DeleteStaleLoginFailures は、止めていない入力ミスの記録のうち auth.LoginFailureRetention の間更新のないものを消し、
// 消した件数を返す（AUTHN-6, ADR 0011）。未登録のアドレスの記録が増え続けないようにする。
func (s *AuthCleanupService) DeleteStaleLoginFailures(ctx context.Context) (int, error) {
	now := s.d.Clock.Now()
	n, err := s.d.Failures.DeleteStale(ctx, now.Add(-auth.LoginFailureRetention), now)
	if err != nil {
		return 0, apperr.Internal(err)
	}
	return n, nil
}
