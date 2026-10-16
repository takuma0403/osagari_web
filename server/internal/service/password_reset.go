package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/auth"
	"github.com/takuma0403/osagari_web/server/internal/clock"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/mail"
	"github.com/takuma0403/osagari_web/server/internal/repository"
)

// PasswordResetDeps は PasswordResetService の依存。
type PasswordResetDeps struct {
	Tx       Transactor
	Users    UserStore
	Codes    VerificationCodeStore
	Tokens   PasswordSetupTokenStore
	Sessions SessionStore
	Devices  TrustedDeviceStore
	Mailer   mail.Mailer
	Clock    clock.Clock
	// Logger は送信数の警告に使う。nil なら slog.Default()
	Logger *slog.Logger
}

// PasswordResetService はメールのコードによるパスワードの再設定（コード送信 → コード確認 → 新しいパスワードの設定）を行う（AUTHN-4, ADR 0010）。
type PasswordResetService struct {
	d PasswordResetDeps
}

func NewPasswordResetService(d PasswordResetDeps) *PasswordResetService {
	d.Logger = loggerOrDefault(d.Logger)
	return &PasswordResetService{d: d}
}

// SendCode はパスワード再設定の認証コードを送る（AUTHN-4）。
//
// 登録の有無を応答や所要時間から判別できないように、未登録・削除済みのアドレスでも登録済みと同じ処理
// （コードの生成・argon2id・保存・メール1通の送信）を行い、メールの本文だけを変える（ADR 0007 と同じ考え）。
// 未登録のアドレスに保存したコードは誰にも届かないので、確認には使えない。
func (s *PasswordResetService) SendCode(ctx context.Context, rawEmail string) error {
	email, _, err := domain.NormalizeStudentEmail(rawEmail)
	if err != nil {
		return errInvalidEmail
	}
	u, err := s.d.Users.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return apperr.Internal(err)
	}
	registered := err == nil && u.Status != domain.UserStatusDeleted
	code, err := issueCode(ctx, s.d.Codes, s.d.Logger, email, domain.VerificationPurposePasswordReset, s.d.Clock.Now())
	if err != nil {
		return err
	}

	msg := passwordResetNotRegisteredMail(email)
	if registered {
		msg = passwordResetCodeMail(email, code)
	}
	if err := s.d.Mailer.Send(ctx, msg); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

// VerifyCode はパスワード再設定のコードを確認し、パスワード設定用の一時トークンを返す（AUTHN-4, AUTHN-6）。
// 入力の回数の数え方は verifyCode を参照。
func (s *PasswordResetService) VerifyCode(ctx context.Context, rawEmail, code string) (string, error) {
	email, _, err := domain.NormalizeStudentEmail(rawEmail)
	if err != nil {
		return "", errInvalidEmail
	}
	now := s.d.Clock.Now()
	if err := verifyCode(ctx, s.d.Codes, email, domain.VerificationPurposePasswordReset, code, now); err != nil {
		return "", err
	}
	token, err := issueSetupToken(ctx, s.d.Tokens, email, domain.PasswordSetupPurposePasswordReset, now)
	if err != nil {
		return "", apperr.Internal(err)
	}
	return token, nil
}

// Complete は一時トークンを使って新しいパスワードを設定し、そのユーザーの既存のセッションをすべて無効にする（AUTHN-4, AUTHN-5）。
// コードで本人確認をしたので最後の確認日時を更新し、deviceID があれば登録済みの端末にして、この端末用のセッションを発行する。
// 停止中・失効なら理由が分かるエラーを返し、パスワードは変えない（トークンも消費しない）。
// トークンの消費・パスワードの更新・セッションの失効と発行・端末の登録は1つのトランザクションで行う。
func (s *PasswordResetService) Complete(ctx context.Context, setupToken, password, deviceID string) (LoggedIn, error) {
	// パスワードの長さが範囲外のときはトークンを消費しない（入力し直せるようにする）
	if err := auth.ValidatePassword(password); err != nil {
		return LoggedIn{}, errPasswordLength
	}
	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return LoggedIn{}, apperr.Internal(err)
	}

	now := s.d.Clock.Now()
	var out LoggedIn
	err = s.d.Tx.Do(ctx, func(ctx context.Context) error {
		t, err := s.d.Tokens.Consume(ctx, auth.HashToken(setupToken), domain.PasswordSetupPurposePasswordReset, now)
		if errors.Is(err, repository.ErrConflict) {
			return errInvalidSetupToken
		}
		if err != nil {
			return err
		}
		u, err := s.d.Users.FindByEmail(ctx, t.Email)
		if errors.Is(err, repository.ErrNotFound) || (err == nil && u.Status == domain.UserStatusDeleted) {
			return errInvalidSetupToken
		}
		if err != nil {
			return err
		}
		// 停止中・失効ならロールバックして、トークンもパスワードもそのままにする
		if err := checkUsable(u, now); err != nil {
			return err
		}
		if err := s.d.Users.UpdatePassword(ctx, u.ID, passwordHash, now); err != nil {
			return err
		}
		if err := s.d.Sessions.RevokeAllByUser(ctx, u.ID, now); err != nil {
			return err
		}
		if err := trustDevice(ctx, s.d.Devices, u.ID, deviceHashOf(deviceID), now); err != nil {
			return err
		}
		sess, err := issueSession(ctx, s.d.Sessions, u.ID, now)
		if err != nil {
			return err
		}
		u.PasswordHash = passwordHash
		u.CodeVerifiedAt = now
		out = LoggedIn{User: u, Session: sess}
		return nil
	})
	if err != nil {
		var ae *apperr.Error
		if errors.As(err, &ae) {
			return LoggedIn{}, ae
		}
		return LoggedIn{}, apperr.Internal(err)
	}
	return out, nil
}
