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

// ExpiredCodeStore は期限切れの認証コードを消す（repository.VerificationCodeRepository）。
type ExpiredCodeStore interface {
	DeleteExpired(ctx context.Context, now, createdBefore time.Time) (int, error)
}

// ExpiredTokenStore は期限切れの一時トークンを消す（repository.LoginChallengeRepository・PasswordSetupTokenRepository）。
type ExpiredTokenStore interface {
	DeleteExpired(ctx context.Context, now time.Time) (int, error)
}

// AuthCleanupDeps は AuthCleanupService の依存。
type AuthCleanupDeps struct {
	Failures StaleLoginFailureStore
	Codes    ExpiredCodeStore
	// Challenges はログインの再確認のチャレンジ（AUTHN-3）
	Challenges ExpiredTokenStore
	// SetupTokens はパスワード設定用の一時トークン（AUTHN-2, AUTHN-4）
	SetupTokens ExpiredTokenStore
	Clock       clock.Clock
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

// DeleteExpiredCodes は期限切れの認証コード・再確認のチャレンジ・パスワード設定用のトークンを消し、消した件数の合計を返す（JOBS-1）。
// 認証コードは送信の制限と全体の送信数に使うため、作ってから auth.CodeRetention の間は消さない（AUTHN-6）。
func (s *AuthCleanupService) DeleteExpiredCodes(ctx context.Context) (int, error) {
	now := s.d.Clock.Now()
	total, err := s.d.Codes.DeleteExpired(ctx, now, now.Add(-auth.CodeRetention))
	if err != nil {
		return 0, apperr.Internal(err)
	}
	for _, st := range []ExpiredTokenStore{s.d.Challenges, s.d.SetupTokens} {
		n, err := st.DeleteExpired(ctx, now)
		if err != nil {
			return total, apperr.Internal(err)
		}
		total += n
	}
	return total, nil
}
