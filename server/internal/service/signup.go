package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"

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
	CodePasswordTooShort        apperr.Code = "password_too_short"
)

var (
	errInvalidEmail = apperr.New(http.StatusBadRequest, CodeInvalidEmail,
		"大学のメールアドレス（学籍番号8桁@edu.cc.saga-u.ac.jp）を入力してください")
	errMalformedCode       = apperr.BadRequest("6桁の数字を入力してください")
	errVerificationExpired = apperr.New(http.StatusBadRequest, CodeVerificationCodeExpired,
		"コードの有効期限が切れたか、使えなくなりました。もう一度コードを送ってください")
	errInvalidSetupToken = apperr.New(http.StatusBadRequest, CodeInvalidSetupToken,
		"手続きの有効期限が切れました。もう一度最初からやり直してください")
	errPasswordTooShort = apperr.New(http.StatusBadRequest, CodePasswordTooShort,
		fmt.Sprintf("パスワードは%d文字以上にしてください", auth.MinPasswordLength))
)

// errWrongCode はコードが違うときのエラー。残りの回数をメッセージに入れる（AUTHN-6）。
func errWrongCode(remaining int) *apperr.Error {
	msg := fmt.Sprintf("コードが違います。あと%d回間違えるとこのコードは使えなくなります", remaining)
	if remaining <= 0 {
		msg = "コードが違います。このコードは使えなくなりました。もう一度コードを送ってください"
	}
	return apperr.New(http.StatusBadRequest, CodeInvalidVerificationCode, msg)
}

var sixDigits = regexp.MustCompile(`^[0-9]{6}$`)

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
}

// SignupService は新規登録（コード送信 → コード確認 → パスワード設定）を行う（AUTH-1, AUTHN-2）。
type SignupService struct {
	d SignupDeps
}

func NewSignupService(d SignupDeps) *SignupService {
	return &SignupService{d: d}
}

// Estimate は大学のメールアドレスから学部・入学年度・有効期限を推定する（AUTH-2〜4）。登録済みかどうかには触れない。
func (s *SignupService) Estimate(rawEmail string) (domain.StudentEstimate, error) {
	_, number, err := domain.NormalizeStudentEmail(rawEmail)
	if err != nil {
		return domain.StudentEstimate{}, errInvalidEmail
	}
	return domain.EstimateStudent(number), nil
}

// SendCode は新規登録の認証コードを送る（AUTHN-2）。
//
// 登録の有無を応答や所要時間から判別できないように、登録済みのアドレスでも未登録と同じ処理
// （コードの生成・argon2id・保存・メール1通の送信）を行い、メールの本文だけを変える。
// 登録済みのアドレスに保存したコードは誰にも届かないので、確認には使えない。
func (s *SignupService) SendCode(ctx context.Context, rawEmail string) error {
	email, _, err := domain.NormalizeStudentEmail(rawEmail)
	if err != nil {
		return errInvalidEmail
	}
	registered, err := s.d.Users.ExistsByEmail(ctx, email)
	if err != nil {
		return apperr.Internal(err)
	}
	code, err := auth.GenerateCode()
	if err != nil {
		return apperr.Internal(err)
	}
	codeHash, err := auth.HashCode(code)
	if err != nil {
		return apperr.Internal(err)
	}

	now := s.d.Clock.Now()
	// 再送したら古いコードは使えなくする（使えるコードは常に1つ）
	if err := s.d.Codes.InvalidateActive(ctx, email, domain.VerificationPurposeSignup, now); err != nil {
		return apperr.Internal(err)
	}
	err = s.d.Codes.Create(ctx, &domain.VerificationCode{
		Email:     email,
		Purpose:   domain.VerificationPurposeSignup,
		CodeHash:  codeHash,
		ExpiresAt: now.Add(auth.CodeTTL),
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return apperr.Internal(err)
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
// 入力は1つのコードにつき auth.MaxCodeFailures 回まで。照合の前に回数を数えるので、同時に送られても上限を超えない。
func (s *SignupService) VerifyCode(ctx context.Context, rawEmail, code string) (VerifiedSignup, error) {
	email, number, err := domain.NormalizeStudentEmail(rawEmail)
	if err != nil {
		return VerifiedSignup{}, errInvalidEmail
	}
	if !sixDigits.MatchString(code) {
		return VerifiedSignup{}, errMalformedCode
	}

	now := s.d.Clock.Now()
	vc, err := s.d.Codes.FindLatestActive(ctx, email, domain.VerificationPurposeSignup, auth.MaxCodeFailures, now)
	if errors.Is(err, repository.ErrNotFound) {
		return VerifiedSignup{}, errVerificationExpired
	}
	if err != nil {
		return VerifiedSignup{}, apperr.Internal(err)
	}
	attempts, err := s.d.Codes.ReserveAttempt(ctx, vc.ID, auth.MaxCodeFailures, now)
	if errors.Is(err, repository.ErrConflict) {
		return VerifiedSignup{}, errVerificationExpired
	}
	if err != nil {
		return VerifiedSignup{}, apperr.Internal(err)
	}
	ok, err := auth.VerifyCode(vc.CodeHash, code)
	if err != nil {
		return VerifiedSignup{}, apperr.Internal(err)
	}
	if !ok {
		return VerifiedSignup{}, errWrongCode(auth.MaxCodeFailures - attempts)
	}
	// 同じコードで同時に確認しても、一時トークンを発行するのは1回だけ
	if err := s.d.Codes.Consume(ctx, vc.ID, now); errors.Is(err, repository.ErrConflict) {
		return VerifiedSignup{}, errVerificationExpired
	} else if err != nil {
		return VerifiedSignup{}, apperr.Internal(err)
	}

	token, err := auth.GenerateToken()
	if err != nil {
		return VerifiedSignup{}, apperr.Internal(err)
	}
	err = s.d.Tokens.Create(ctx, &domain.PasswordSetupToken{
		Email:     email,
		Purpose:   domain.PasswordSetupPurposeSignup,
		TokenHash: auth.HashToken(token),
		ExpiresAt: now.Add(auth.SetupTokenTTL),
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return VerifiedSignup{}, apperr.Internal(err)
	}
	return VerifiedSignup{SetupToken: token, Estimate: domain.EstimateStudent(number)}, nil
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
	// パスワードが短いときはトークンを消費しない（入力し直せるようにする）
	if err := auth.ValidatePassword(password); err != nil {
		return CompletedSignup{}, errPasswordTooShort
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
		_, number, err := domain.NormalizeStudentEmail(t.Email)
		if err != nil {
			return err
		}
		est := domain.EstimateStudent(number)
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
		if err := trustDevice(ctx, s.d.Devices, u.ID, deviceID, now); err != nil {
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
