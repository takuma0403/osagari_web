// Package service は業務ロジックを置く。repository・mail・clock はこのパッケージのインターフェース越しに使い、
// エラーは apperr で返す（handler はそのまま return する）。
package service

import (
	"context"
	"log/slog"
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
	FindByID(ctx context.Context, id string) (*domain.User, error)
	FindByIDs(ctx context.Context, ids []string) (map[string]*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	UpdateCodeVerifiedAt(ctx context.Context, id string, at time.Time) error
	UpdatePassword(ctx context.Context, id, passwordHash string, at time.Time) error
	UpdateSellerStatus(ctx context.Context, id string, from []domain.SellerStatus, to domain.SellerStatus, at time.Time) error
}

// VerificationCodeStore は verification_codes の読み書き（repository.VerificationCodeRepository）。
type VerificationCodeStore interface {
	CreateWithinLimit(ctx context.Context, c *domain.VerificationCode, minInterval, window time.Duration, maxInWindow int) error
	CountByEmailSince(ctx context.Context, email string, since time.Time) (int, error)
	CountSince(ctx context.Context, since time.Time) (int, error)
	InvalidateOthers(ctx context.Context, email string, purpose domain.VerificationPurpose, keepID string, now time.Time) error
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
	FindActiveByTokenHash(ctx context.Context, tokenHash string, now time.Time) (*domain.Session, error)
	Revoke(ctx context.Context, tokenHash string, now time.Time) error
	RevokeAllByUser(ctx context.Context, userID string, now time.Time) error
}

// LoginFailureStore は login_failures の読み書き（repository.LoginFailureRepository）。
type LoginFailureStore interface {
	ReserveAttempt(ctx context.Context, email string, maxAttempts int, lockFor time.Duration, now time.Time) error
	Clear(ctx context.Context, email string, now time.Time) error
	Reset(ctx context.Context, email string, now time.Time) error
}

// TrustedDeviceStore は trusted_devices の読み書き（repository.TrustedDeviceRepository）。
type TrustedDeviceStore interface {
	Upsert(ctx context.Context, d *domain.TrustedDevice) error
	Exists(ctx context.Context, userID, deviceHash string) (bool, error)
}

// LoginChallengeStore は login_challenges の読み書き（repository.LoginChallengeRepository）。
type LoginChallengeStore interface {
	Create(ctx context.Context, c *domain.LoginChallenge) error
	FindActive(ctx context.Context, tokenHash string, now time.Time) (*domain.LoginChallenge, error)
	Consume(ctx context.Context, id string, now time.Time) error
}

// StudentCardStore は student_cards の読み書き（repository.StudentCardRepository）。
type StudentCardStore interface {
	Create(ctx context.Context, c *domain.StudentCard) error
	FindCurrentByUser(ctx context.Context, userID string) (*domain.StudentCard, error)
	ExistsByObjectKey(ctx context.Context, objectKey string) (bool, error)
	SupersedeCurrent(ctx context.Context, userID string, now time.Time) (bool, error)
	FindByID(ctx context.Context, id string) (*domain.StudentCard, error)
	ListPending(ctx context.Context) ([]*domain.StudentCard, error)
	Review(ctx context.Context, id string, to domain.StudentCardStatus, adminUserID string, now time.Time) error
}

// StudentCardViewStore は student_card_views の読み書き（repository.StudentCardViewRepository）。
type StudentCardViewStore interface {
	Create(ctx context.Context, v *domain.StudentCardView) error
	ListByCard(ctx context.Context, studentCardID string) ([]*domain.StudentCardView, error)
}

// ItemStore は items・item_images の読み書き（repository.ItemRepository）。
type ItemStore interface {
	Create(ctx context.Context, it *domain.Item) error
	FindByID(ctx context.Context, id string) (*domain.Item, error)
	UpdateContent(ctx context.Context, id, sellerID string, status domain.ItemStatus, updatedAt time.Time, c domain.ItemContent, now time.Time) error
	UpdateStatus(ctx context.Context, id, sellerID string, from []domain.ItemStatus, unchangedSince *time.Time, to domain.ItemStatus, now time.Time) error
	UpdateDealStatus(ctx context.Context, id, sellerID string, from []domain.ItemStatus, to domain.ItemStatus, now time.Time) error
	ReplaceImages(ctx context.Context, itemID string, images []domain.ItemImage, now time.Time) error
	FindByShareID(ctx context.Context, shareID string) (*domain.Item, error)
	ListPublic(ctx context.Context, f domain.ItemFilter, after *domain.ItemCursor, limit int, now time.Time) ([]*domain.Item, int, error)
	FindByIDs(ctx context.Context, ids []string) (map[string]*domain.Item, error)
	ListBySeller(ctx context.Context, sellerID string) ([]*domain.Item, error)
	ListForAdmin(ctx context.Context, after *domain.AdminCursor, limit int) ([]*domain.Item, error)
}

// TransactionStore は transactions の読み書き（repository.TransactionRepository）。
type TransactionStore interface {
	Create(ctx context.Context, tr *domain.Transaction) error
	FindByID(ctx context.Context, id string) (*domain.Transaction, error)
	ListRequestedByItem(ctx context.Context, itemID string) ([]*domain.Transaction, error)
	ListExpiredRequests(ctx context.Context, now time.Time) ([]*domain.Transaction, error)
	CountRequestedByItem(ctx context.Context, itemID string) (int, error)
	FindRequestedByItemAndBuyer(ctx context.Context, itemID, buyerID string) (*domain.Transaction, error)
	CountCompletedByUsers(ctx context.Context, userIDs []string) (map[string]int, error)
	CountCancelledByUsers(ctx context.Context, userIDs []string) (map[string]int, error)
	CountInDealByUsers(ctx context.Context, userIDs []string) (map[string]int, error)
	ListForAdmin(ctx context.Context, after *domain.AdminCursor, limit int) ([]*domain.Transaction, error)
	ListByParty(ctx context.Context, role domain.TransactionRole, userID string, statuses []domain.TransactionStatus, after *domain.TransactionCursor, limit int) ([]*domain.Transaction, error)
	ListRequestGroupsBySeller(ctx context.Context, sellerID string) ([]domain.TransactionRequestGroup, error)
	ListByPartyAndStatus(ctx context.Context, userID string, status domain.TransactionStatus) ([]*domain.Transaction, error)
	Approve(ctx context.Context, id, sellerID string, feeRateBP int, fee, proceeds int64, now time.Time) error
	UpdateStatus(ctx context.Context, id string, from, to domain.TransactionStatus, now time.Time) error
	Acknowledge(ctx context.Context, id, sellerID string, now time.Time) error
	Complete(ctx context.Context, id, buyerID string, method domain.HandoverMethod, now time.Time) error
	AgreeCancel(ctx context.Context, id, userID string, role domain.TransactionRole, now time.Time) error
	Cancel(ctx context.Context, id string, now time.Time) error
	CancelBySystem(ctx context.Context, id string, now time.Time) error
	WithdrawCancel(ctx context.Context, id, userID string, role domain.TransactionRole, now time.Time) error
}

// HandoverTokenStore は handover_tokens の読み書き（repository.HandoverTokenRepository）。
type HandoverTokenStore interface {
	Create(ctx context.Context, t *domain.HandoverToken) error
	InvalidateOthers(ctx context.Context, transactionID string, kind domain.HandoverTokenKind, keepID string, now time.Time) error
	FindActiveByTokenHash(ctx context.Context, transactionID, tokenHash string, now time.Time) (*domain.HandoverToken, error)
	ListActiveWithCode(ctx context.Context, transactionID string, now time.Time) ([]*domain.HandoverToken, error)
	Consume(ctx context.Context, id string, now time.Time) error
	DeleteExpired(ctx context.Context, now time.Time) (int, error)
}

// PaymentStore は payments の読み出し（repository.PaymentRepository）。記録は payment.Gateway 越しに行う。
type PaymentStore interface {
	ListPayoutsByTransactions(ctx context.Context, transactionIDs []string) (map[string]*domain.Payment, error)
}

// SettingStore は settings の読み書き（repository.SettingRepository）。
type SettingStore interface {
	FeeRate(ctx context.Context) (int, error)
}

// AdminSettingStore は管理画面での手数料率の読み書き（repository.SettingRepository。ADR 0034）。
type AdminSettingStore interface {
	FeeRateSetting(ctx context.Context) (domain.FeeRateSetting, error)
	UpdateFeeRate(ctx context.Context, bp int, adminUserID string, now time.Time) error
}

// AdminUserStore は管理画面でのユーザーの読み書き（repository.UserRepository。ADR 0034）。
type AdminUserStore interface {
	FindByID(ctx context.Context, id string) (*domain.User, error)
	ListForAdmin(ctx context.Context, f domain.AdminUserQuery, after *domain.AdminCursor, limit int) ([]*domain.User, error)
	CountWithdrawalRequested(ctx context.Context) (int, error)
	UpdateStatus(ctx context.Context, id string, from []domain.UserStatus, to domain.UserStatus, now time.Time) error
	UpdateExpiresOn(ctx context.Context, id string, from []domain.UserStatus, expiresOn, now time.Time) error
}

// loggerOrDefault は l が nil なら slog.Default() を返す。
func loggerOrDefault(l *slog.Logger) *slog.Logger {
	if l == nil {
		return slog.Default()
	}
	return l
}
