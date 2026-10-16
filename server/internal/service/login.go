package service

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/auth"
	"github.com/takuma0403/osagari_web/server/internal/clock"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/mail"
	"github.com/takuma0403/osagari_web/server/internal/repository"
)

// ログインの機能固有のエラーコード（docs/tech/api.md の API-1、ADR 0009）。
const (
	CodeInvalidCredentials    apperr.Code = "invalid_credentials"
	CodeAccountSuspended      apperr.Code = "account_suspended"
	CodeAccountExpired        apperr.Code = "account_expired"
	CodeInvalidLoginChallenge apperr.Code = "invalid_login_challenge"
)

var (
	// メールアドレスの形式違い・未登録・削除済み・パスワード違いを区別しない（登録の有無を判別させない。AUTHN-2）
	errInvalidCredentials = apperr.New(http.StatusUnauthorized, CodeInvalidCredentials,
		"メールアドレスかパスワードが違います")
	errAccountSuspended = apperr.New(http.StatusForbidden, CodeAccountSuspended,
		"このアカウントは利用停止中のため、ログインできません")
	errAccountExpired = apperr.New(http.StatusForbidden, CodeAccountExpired,
		"このアカウントは有効期限が切れたため、ログインできません")
	errInvalidLoginChallenge = apperr.New(http.StatusBadRequest, CodeInvalidLoginChallenge,
		"ログインの手続きの有効期限が切れました。もう一度ログインしてください")
)

// LoginDeps は LoginService の依存。
type LoginDeps struct {
	Tx         Transactor
	Users      UserStore
	Codes      VerificationCodeStore
	Challenges LoginChallengeStore
	Sessions   SessionStore
	Devices    TrustedDeviceStore
	Mailer     mail.Mailer
	Clock      clock.Clock
	// Logger は送信数の警告に使う。nil なら slog.Default()
	Logger *slog.Logger
}

// LoginService はパスワードでのログインと、メールのコードによる再確認を行う（AUTH-1, AUTHN-3, AUTHN-5, ADR 0009）。
type LoginService struct {
	d LoginDeps
}

func NewLoginService(d LoginDeps) *LoginService {
	d.Logger = loggerOrDefault(d.Logger)
	return &LoginService{d: d}
}

// IssuedLoginChallenge は発行したログインチャレンジ。Token はコードの確認・再送に使う値で、DB にはハッシュだけを保存している。
type IssuedLoginChallenge struct {
	Token  string
	Reason domain.LoginRecheckReason
	// Email はコードの送信先
	Email string
}

// LoginResult はログインの結果。再確認が要らなければ Session、要るなら Challenge のどちらか一方が入る。
type LoginResult struct {
	User      *domain.User
	Session   *IssuedSession
	Challenge *IssuedLoginChallenge
}

// LoggedIn はコードの確認まで済んでログインした結果。
type LoggedIn struct {
	User    *domain.User
	Session IssuedSession
}

// Login はメールアドレスとパスワードでログインする（AUTHN-3）。
//
// パスワードが一致し、最後にコードで確認してから auth.RecheckInterval 未満で、端末（deviceID）が登録済みなら
// そのままセッションを発行する。そうでなければメールにコードを送り、再確認用のチャレンジを返す。
func (s *LoginService) Login(ctx context.Context, rawEmail, password, deviceID string) (LoginResult, error) {
	// 形式の違うアドレスと長さが範囲外のパスワードは、登録済みのユーザーと一致しえないので照合しない
	email, _, err := domain.NormalizeStudentEmail(rawEmail)
	if err != nil || auth.ValidatePassword(password) != nil {
		return LoginResult{}, errInvalidCredentials
	}
	u, err := s.d.Users.FindByEmail(ctx, email)
	if errors.Is(err, repository.ErrNotFound) || (err == nil && u.Status == domain.UserStatusDeleted) {
		// 未登録・削除済みでも同じだけ計算して、所要時間から判別させない
		auth.VerifyDummyPassword(password)
		return LoginResult{}, errInvalidCredentials
	}
	if err != nil {
		return LoginResult{}, apperr.Internal(err)
	}
	ok, err := auth.VerifyPassword(u.PasswordHash, password)
	if err != nil {
		return LoginResult{}, apperr.Internal(err)
	}
	if !ok {
		return LoginResult{}, errInvalidCredentials
	}

	now := s.d.Clock.Now()
	if err := checkUsable(u, now); err != nil {
		return LoginResult{}, err
	}
	deviceHash := deviceHashOf(deviceID)
	reason, err := s.recheckReason(ctx, u, deviceHash, now)
	if err != nil {
		return LoginResult{}, apperr.Internal(err)
	}
	if reason != "" {
		c, err := s.startChallenge(ctx, u, reason, deviceHash, now)
		if err != nil {
			return LoginResult{}, err
		}
		return LoginResult{User: u, Challenge: &c}, nil
	}

	var sess IssuedSession
	err = s.d.Tx.Do(ctx, func(ctx context.Context) error {
		if err := trustDevice(ctx, s.d.Devices, u.ID, deviceHash, now); err != nil {
			return err
		}
		sess, err = issueSession(ctx, s.d.Sessions, u.ID, now)
		return err
	})
	if err != nil {
		return LoginResult{}, apperr.Internal(err)
	}
	return LoginResult{User: u, Session: &sess}, nil
}

// recheckReason は再確認が必要な理由を返す。要らなければ空。両方に当てはまるときは code_expired（ADR 0009）。
func (s *LoginService) recheckReason(ctx context.Context, u *domain.User, deviceHash string, now time.Time) (domain.LoginRecheckReason, error) {
	if now.Sub(u.CodeVerifiedAt) >= auth.RecheckInterval {
		return domain.LoginRecheckReasonCodeExpired, nil
	}
	if deviceHash == "" {
		return domain.LoginRecheckReasonNewDevice, nil
	}
	known, err := s.d.Devices.Exists(ctx, u.ID, deviceHash)
	if err != nil {
		return "", err
	}
	if !known {
		return domain.LoginRecheckReasonNewDevice, nil
	}
	return "", nil
}

// startChallenge はチャレンジを発行し、コードをメールで送る。ログイン時の端末はチャレンジに保存し、確認に成功したら登録する。
func (s *LoginService) startChallenge(ctx context.Context, u *domain.User, reason domain.LoginRecheckReason, deviceHash string, now time.Time) (IssuedLoginChallenge, error) {
	token, err := auth.GenerateToken()
	if err != nil {
		return IssuedLoginChallenge{}, apperr.Internal(err)
	}
	err = s.d.Challenges.Create(ctx, &domain.LoginChallenge{
		UserID:     u.ID,
		TokenHash:  auth.HashToken(token),
		Reason:     reason,
		DeviceHash: deviceHash,
		ExpiresAt:  now.Add(auth.LoginChallengeTTL),
		CreatedAt:  now,
		UpdatedAt:  now,
	})
	if err != nil {
		return IssuedLoginChallenge{}, apperr.Internal(err)
	}
	if err := s.sendRecheckCode(ctx, u.Email, now); err != nil {
		return IssuedLoginChallenge{}, err
	}
	return IssuedLoginChallenge{Token: token, Reason: reason, Email: u.Email}, nil
}

// sendRecheckCode は再確認のコードを作り直してメールで送る。古いコードは使えなくなる。
func (s *LoginService) sendRecheckCode(ctx context.Context, email string, now time.Time) error {
	code, err := issueCode(ctx, s.d.Codes, s.d.Logger, email, domain.VerificationPurposeLoginRecheck, now)
	if err != nil {
		return err
	}
	if err := s.d.Mailer.Send(ctx, loginRecheckCodeMail(email, code)); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

// ResendRecheckCode はチャレンジが使えるなら、再確認のコードを送り直す（AUTHN-3）。送信回数の制限は issueCode（AUTHN-6）。
func (s *LoginService) ResendRecheckCode(ctx context.Context, challengeToken string) error {
	now := s.d.Clock.Now()
	_, u, err := s.activeChallenge(ctx, challengeToken, now)
	if err != nil {
		return err
	}
	return s.sendRecheckCode(ctx, u.Email, now)
}

// VerifyRecheck は再確認のコードを確認してログインする（AUTHN-3, AUTHN-6）。
// 成功したら、チャレンジの消費・最後の確認日時の更新・ログイン時の端末の登録・セッションの発行を1つのトランザクションで行う。
func (s *LoginService) VerifyRecheck(ctx context.Context, challengeToken, code string) (LoggedIn, error) {
	now := s.d.Clock.Now()
	c, u, err := s.activeChallenge(ctx, challengeToken, now)
	if err != nil {
		return LoggedIn{}, err
	}
	if err := verifyCode(ctx, s.d.Codes, u.Email, domain.VerificationPurposeLoginRecheck, code, now); err != nil {
		return LoggedIn{}, err
	}

	var sess IssuedSession
	err = s.d.Tx.Do(ctx, func(ctx context.Context) error {
		if err := s.d.Challenges.Consume(ctx, c.ID, now); errors.Is(err, repository.ErrConflict) {
			return errInvalidLoginChallenge
		} else if err != nil {
			return err
		}
		if err := s.d.Users.UpdateCodeVerifiedAt(ctx, u.ID, now); err != nil {
			return err
		}
		if err := trustDevice(ctx, s.d.Devices, u.ID, c.DeviceHash, now); err != nil {
			return err
		}
		sess, err = issueSession(ctx, s.d.Sessions, u.ID, now)
		return err
	})
	if err != nil {
		var ae *apperr.Error
		if errors.As(err, &ae) {
			return LoggedIn{}, ae
		}
		return LoggedIn{}, apperr.Internal(err)
	}
	u.CodeVerifiedAt = now
	return LoggedIn{User: u, Session: sess}, nil
}

// activeChallenge は使えるチャレンジとそのユーザーを返す。チャレンジが使えないか、ユーザーが利用できなくなっていればエラー。
func (s *LoginService) activeChallenge(ctx context.Context, challengeToken string, now time.Time) (*domain.LoginChallenge, *domain.User, error) {
	if challengeToken == "" {
		return nil, nil, errInvalidLoginChallenge
	}
	c, err := s.d.Challenges.FindActive(ctx, auth.HashToken(challengeToken), now)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, nil, errInvalidLoginChallenge
	}
	if err != nil {
		return nil, nil, apperr.Internal(err)
	}
	u, err := s.d.Users.FindByID(ctx, c.UserID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, nil, errInvalidLoginChallenge
	}
	if err != nil {
		return nil, nil, apperr.Internal(err)
	}
	if u.Status == domain.UserStatusDeleted {
		return nil, nil, errInvalidLoginChallenge
	}
	if err := checkUsable(u, now); err != nil {
		return nil, nil, err
	}
	return c, u, nil
}

// checkUsable は、パスワードが一致したユーザーがログインできる状態かを確かめる（ADR 0009）。
// 停止中・失効（有効期限切れを含む）なら理由が分かるエラーを返す。
func checkUsable(u *domain.User, now time.Time) error {
	switch {
	case u.Status == domain.UserStatusSuspended:
		return errAccountSuspended
	case u.IsExpired(now):
		return errAccountExpired
	case u.Status != domain.UserStatusActive:
		return errInvalidCredentials
	}
	return nil
}
