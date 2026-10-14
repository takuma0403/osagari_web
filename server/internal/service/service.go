// Package service は業務ロジックを置く。repository・mail・clock はこのパッケージのインターフェース越しに使い、
// エラーは apperr で返す（handler はそのまま return する）。
package service

import (
	"context"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/domain"
)

// Transactor は複数のテーブルにまたがる処理を1つのトランザクションで行う（repository.Transactor）。
// fn に渡される ctx を各 store に渡すと、同じトランザクションで実行される。
type Transactor interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

// UserStore は users の読み書き（repository.UserRepository）。
type UserStore interface {
	Create(ctx context.Context, u *domain.User) error
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}

// VerificationCodeStore は verification_codes の読み書き（repository.VerificationCodeRepository）。
type VerificationCodeStore interface {
	Create(ctx context.Context, c *domain.VerificationCode) error
	InvalidateActive(ctx context.Context, email string, purpose domain.VerificationPurpose, now time.Time) error
	FindLatestActive(ctx context.Context, email string, purpose domain.VerificationPurpose, maxAttempts int, now time.Time) (*domain.VerificationCode, error)
	ReserveAttempt(ctx context.Context, id string, maxAttempts int, now time.Time) (int, error)
	Consume(ctx context.Context, id string, now time.Time) error
}

// PasswordSetupTokenStore は password_setup_tokens の読み書き（repository.PasswordSetupTokenRepository）。
type PasswordSetupTokenStore interface {
	Create(ctx context.Context, t *domain.PasswordSetupToken) error
	Consume(ctx context.Context, tokenHash string, purpose domain.PasswordSetupPurpose, now time.Time) (*domain.PasswordSetupToken, error)
}

// SessionStore は sessions の読み書き（repository.SessionRepository）。
type SessionStore interface {
	Create(ctx context.Context, s *domain.Session) error
}

// TrustedDeviceStore は trusted_devices の読み書き（repository.TrustedDeviceRepository）。
type TrustedDeviceStore interface {
	Upsert(ctx context.Context, d *domain.TrustedDevice) error
}
