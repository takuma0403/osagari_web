package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/auth"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/repository"
)

// メールで送る6桁の認証コードの発行と照合（AUTHN-2, AUTHN-3, AUTHN-6）。新規登録とログインの再確認で共通に使う。

var (
	errMalformedCode       = apperr.BadRequest("6桁の数字を入力してください")
	errVerificationExpired = apperr.New(http.StatusBadRequest, CodeVerificationCodeExpired,
		"コードの有効期限が切れたか、使えなくなりました。もう一度コードを送ってください")
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

// issueCode は6桁のコードを生成し、argon2id のハッシュを保存して、平文のコードを返す（有効期限は auth.CodeTTL）。
// 同じアドレス・用途の古いコードは使えなくする（使えるコードは常に1つ）。メールは呼び出し側が送る。
func issueCode(ctx context.Context, codes VerificationCodeStore, email string, purpose domain.VerificationPurpose, now time.Time) (string, error) {
	code, err := auth.GenerateCode()
	if err != nil {
		return "", err
	}
	codeHash, err := auth.HashCode(code)
	if err != nil {
		return "", err
	}
	if err := codes.InvalidateActive(ctx, email, purpose, now); err != nil {
		return "", err
	}
	err = codes.Create(ctx, &domain.VerificationCode{
		Email:     email,
		Purpose:   purpose,
		CodeHash:  codeHash,
		ExpiresAt: now.Add(auth.CodeTTL),
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return "", err
	}
	return code, nil
}

// verifyCode は同じアドレス・用途のいちばん新しい使えるコードと照合し、一致したら使用済みにする（AUTHN-6）。
// 入力は1つのコードにつき auth.MaxCodeFailures 回まで。照合の前に回数を数えるので、同時に送られても上限を超えない。
// 同じコードで同時に確認しても、成功するのは1回だけ。エラーは apperr で返す。
func verifyCode(ctx context.Context, codes VerificationCodeStore, email string, purpose domain.VerificationPurpose, code string, now time.Time) error {
	if !sixDigits.MatchString(code) {
		return errMalformedCode
	}
	vc, err := codes.FindLatestActive(ctx, email, purpose, auth.MaxCodeFailures, now)
	if errors.Is(err, repository.ErrNotFound) {
		return errVerificationExpired
	}
	if err != nil {
		return apperr.Internal(err)
	}
	attempts, err := codes.ReserveAttempt(ctx, vc.ID, auth.MaxCodeFailures, now)
	if errors.Is(err, repository.ErrConflict) {
		return errVerificationExpired
	}
	if err != nil {
		return apperr.Internal(err)
	}
	ok, err := auth.VerifyCode(vc.CodeHash, code)
	if err != nil {
		return apperr.Internal(err)
	}
	if !ok {
		return errWrongCode(auth.MaxCodeFailures - attempts)
	}
	if err := codes.Consume(ctx, vc.ID, now); errors.Is(err, repository.ErrConflict) {
		return errVerificationExpired
	} else if err != nil {
		return apperr.Internal(err)
	}
	return nil
}
