package service

import (
	"context"
	"errors"
	"fmt"
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

// 新規登録の機能固有のエラーコード（docs/tech/api.md の API-1）。
const (
	CodeInvalidEmail            apperr.Code = "invalid_email"
	CodeInvalidVerificationCode apperr.Code = "invalid_verification_code"
	CodeVerificationCodeExpired apperr.Code = "verification_code_expired"
	CodeInvalidSetupToken       apperr.Code = "invalid_setup_token"
	CodeInvalidPasswordLength   apperr.Code = "invalid_password_length"
	CodeSignupPeriodEnded       apperr.Code = "signup_period_ended"
)

var (
	errInvalidEmail = apperr.New(http.StatusBadRequest, CodeInvalidEmail,
		"大学のメールアドレス（学籍番号8桁@edu.cc.saga-u.ac.jp）を入力してください")
	errInvalidSetupToken = apperr.New(http.StatusBadRequest, CodeInvalidSetupToken,
		"手続きの有効期限が切れました。もう一度最初からやり直してください")
	errPasswordLength = apperr.New(http.StatusBadRequest, CodeInvalidPasswordLength,
		fmt.Sprintf("パスワードは%d文字以上%d文字以下にしてください", auth.MinPasswordLength, auth.MaxPasswordLength))
	errSignupPeriodEnded = apperr.New(http.StatusBadRequest, CodeSignupPeriodEnded,
		"この学籍番号は、登録できる期間を過ぎています")
)

// SignupDeps は SignupService の依存。
type SignupDeps struct {
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

// SignupService は新規登録（コード送信 → コード確認 → パスワード設定）を行う（AUTH-1, AUTHN-2）。
type SignupService struct {
	d SignupDeps
}

func NewSignupService(d SignupDeps) *SignupService {
	d.Logger = loggerOrDefault(d.Logger)
	return &SignupService{d: d}
}

// Estimate は大学のメールアドレスから学部・入学年度・有効期限を推定する（AUTH-2〜4）。登録済みかどうかには触れない。
// 有効期限を過ぎた学籍番号は signup_period_ended にする（AUTH-3, ADR 0047）。
func (s *SignupService) Estimate(rawEmail string) (domain.StudentEstimate, error) {
	_, est, err := estimateSignup(rawEmail, s.d.Clock.Now())
	return est, err
}

// estimateSignup は大学のメールアドレスを正規化し、学籍番号から推定した情報を返す（AUTH-2〜4）。
// 形式が違えば invalid_email、有効期限を過ぎていれば signup_period_ended を返す（AUTH-1, AUTH-3, ADR 0047）。
// 学籍番号だけで決まるので、登録の有無を調べる前に呼ぶ（AUTHN-2）。
func estimateSignup(rawEmail string, now time.Time) (string, domain.StudentEstimate, error) {
	email, number, err := domain.NormalizeStudentEmail(rawEmail)
	if err != nil {
		return "", domain.StudentEstimate{}, errInvalidEmail
	}
	est := domain.EstimateStudent(number)
	if est.IsExpired(now) {
		return "", domain.StudentEstimate{}, errSignupPeriodEnded
	}
	return email, est, nil
}

// SendCode は新規登録の認証コードを送る（AUTHN-2）。
//
// 登録の有無を応答や所要時間から判別できないように、登録済みのアドレスでも未登録と同じ処理
// （コードの生成・argon2id・保存・メール1通の送信）を行い、メールの本文だけを変える。
// 登録済みのアドレスに保存したコードは誰にも届かないので、確認には使えない。
func (s *SignupService) SendCode(ctx context.Context, rawEmail string) error {
	now := s.d.Clock.Now()
	email, _, err := estimateSignup(rawEmail, now)
	if err != nil {
		return err
	}
	registered, err := s.d.Users.ExistsByEmail(ctx, email)
	if err != nil {
		return apperr.Internal(err)
	}
	// 登録済みでも同じようにコードを作って保存する（所要時間とその後の確認の応答を揃える。ADR 0007）
	code, err := issueCode(ctx, s.d.Codes, s.d.Logger, email, domain.VerificationPurposeSignup, now)
	if err != nil {
		return err
	}

	msg := signupCodeMail(email, code)
	if registered {
		msg = signupAlreadyRegisteredMail(email)
	}
	if err := s.d.Mailer.Send(ctx, msg); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

// VerifiedSignup はコードの確認結果。SetupToken はパスワード設定用の一時トークン（15分有効・一度だけ使える）。
type VerifiedSignup struct {
	SetupToken string
	Estimate   domain.StudentEstimate
}

// VerifyCode は新規登録のコードを確認し、パスワード設定用の一時トークンを発行する（AUTHN-2, AUTHN-6）。
// 入力の回数の数え方は verifyCode を参照。
func (s *SignupService) VerifyCode(ctx context.Context, rawEmail, code string) (VerifiedSignup, error) {
	now := s.d.Clock.Now()
	email, est, err := estimateSignup(rawEmail, now)
	if err != nil {
		return VerifiedSignup{}, err
	}
	if err := verifyCode(ctx, s.d.Codes, email, domain.VerificationPurposeSignup, code, now); err != nil {
		return VerifiedSignup{}, err
	}

	token, err := issueSetupToken(ctx, s.d.Tokens, email, domain.PasswordSetupPurposeSignup, now)
	if err != nil {
		return VerifiedSignup{}, apperr.Internal(err)
	}
	return VerifiedSignup{SetupToken: token, Estimate: est}, nil
}

// issueSetupToken はパスワード設定用の一時トークン（auth.SetupTokenTTL 有効・一度だけ使える）を発行し、
// SHA-256 のハッシュを保存して平文のトークンを返す（AUTHN-2, AUTHN-4, ADR 0007）。
func issueSetupToken(ctx context.Context, tokens PasswordSetupTokenStore, email string, purpose domain.PasswordSetupPurpose, now time.Time) (string, error) {
	token, err := auth.GenerateToken()
	if err != nil {
		return "", err
	}
	err = tokens.Create(ctx, &domain.PasswordSetupToken{
		Email:     email,
		Purpose:   purpose,
		TokenHash: auth.HashToken(token),
		ExpiresAt: now.Add(auth.SetupTokenTTL),
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return "", err
	}
	return token, nil
}

// CompletedSignup は登録の結果。作成したユーザーと、ログイン状態にするためのセッション。
type CompletedSignup struct {
	User    *domain.User
	Session IssuedSession
}

// Complete は一時トークンを使ってパスワードを設定し、アカウントを作ってログイン状態にする（AUTHN-2, AUTH-2〜4, AUTHN-5）。
// 学籍番号から入学年度・学部・有効期限を計算して登録する。deviceID があれば登録済みの端末にする（AUTHN-3）。
// トークンの消費・ユーザー・セッション・端末の作成は1つのトランザクションで行う。
func (s *SignupService) Complete(ctx context.Context, setupToken, password, deviceID string) (CompletedSignup, error) {
	// パスワードの長さが範囲外のときはトークンを消費しない（入力し直せるようにする）
	if err := auth.ValidatePassword(password); err != nil {
		return CompletedSignup{}, errPasswordLength
	}
	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return CompletedSignup{}, apperr.Internal(err)
	}

	now := s.d.Clock.Now()
	var out CompletedSignup
	err = s.d.Tx.Do(ctx, func(ctx context.Context) error {
		t, err := s.d.Tokens.Consume(ctx, auth.HashToken(setupToken), domain.PasswordSetupPurposeSignup, now)
		if errors.Is(err, repository.ErrConflict) {
			return errInvalidSetupToken
		}
		if err != nil {
			return err
		}
		// コードの確認からここまでの間に有効期限を過ぎた場合も断る。エラーでトランザクションを戻すので、トークンは消費しない
		_, est, err := estimateSignup(t.Email, now)
		if err != nil {
			return err
		}
		u := &domain.User{
			Email:          t.Email,
			StudentNumber:  est.StudentNumber,
			AdmissionYear:  est.AdmissionYear,
			DepartmentCode: est.DepartmentCode,
			ExpiresOn:      est.ExpiresOn,
			Status:         domain.UserStatusActive,
			SellerStatus:   domain.SellerStatusNone,
			PasswordHash:   passwordHash,
			CodeVerifiedAt: now,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if est.Department != nil {
			u.Department = &est.Department.Code
		}
		// 登録済みのアドレスは、ここまで来ても（コードを当てても）作らない。応答は無効なトークンと同じにする
		if err := s.d.Users.Create(ctx, u); errors.Is(err, repository.ErrDuplicate) {
			return errInvalidSetupToken
		} else if err != nil {
			return err
		}
		sess, err := issueSession(ctx, s.d.Sessions, u.ID, now)
		if err != nil {
			return err
		}
		if err := trustDevice(ctx, s.d.Devices, u.ID, deviceHashOf(deviceID), now); err != nil {
			return err
		}
		out = CompletedSignup{User: u, Session: sess}
		return nil
	})
	if err != nil {
		var ae *apperr.Error
		if errors.As(err, &ae) {
			return CompletedSignup{}, ae
		}
		return CompletedSignup{}, apperr.Internal(err)
	}
	return out, nil
}
