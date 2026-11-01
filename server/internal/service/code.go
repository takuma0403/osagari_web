package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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

// errCodeSentRecently・errTooManyCodeSends は同じアドレスへのコード送信が上限に達したときのエラー（AUTHN-6）。
// 登録済みかどうかにかかわらず同じように数えるので、この応答から登録の有無は分からない。
var (
	errCodeSentRecently = apperr.TooManyRequests("コードを送ったばかりです。1分ほどたってからもう一度お試しください")
	errTooManyCodeSends = apperr.TooManyRequests("コードの送信が続いたため、しばらく送れません。1時間ほどたってからもう一度お試しください")
)

// issueCode は6桁のコードを生成し、argon2id のハッシュを保存して、平文のコードを返す（有効期限は auth.CodeTTL）。
// 同じアドレスへの送信は用途をまとめて auth.CodeSendInterval に1回・auth.CodeSendWindow に auth.MaxCodeSends 回までで、
// 超えたら保存せずに 429 を返す（AUTHN-6, ADR 0011）。上限で断ったときは、それまでのコードはそのまま使える。
// 保存できたら同じアドレス・用途の古いコードは使えなくする（使えるコードは常に1つ）。メールは呼び出し側が送る。
// エラーは apperr で返す。
func issueCode(ctx context.Context, codes VerificationCodeStore, logger *slog.Logger, email string, purpose domain.VerificationPurpose, now time.Time) (string, error) {
	code, err := auth.GenerateCode()
	if err != nil {
		return "", apperr.Internal(err)
	}
	codeHash, err := auth.HashCode(code)
	if err != nil {
		return "", apperr.Internal(err)
	}
	vc := &domain.VerificationCode{
		Email:     email,
		Purpose:   purpose,
		CodeHash:  codeHash,
		ExpiresAt: now.Add(auth.CodeTTL),
		CreatedAt: now,
		UpdatedAt: now,
	}
	err = codes.CreateWithinLimit(ctx, vc, auth.CodeSendInterval, auth.CodeSendWindow, auth.MaxCodeSends)
	if errors.Is(err, repository.ErrRateLimited) {
		return "", codeSendLimitError(ctx, codes, email, now)
	}
	if err != nil {
		return "", apperr.Internal(err)
	}
	if err := codes.InvalidateOthers(ctx, email, purpose, vc.ID, now); err != nil {
		return "", apperr.Internal(err)
	}
	warnIfManyMails(ctx, codes, logger, now)
	return code, nil
}

// codeSendLimitError は、送信を断った理由（間隔が短い・回数が多い）に合わせたエラーを返す。
func codeSendLimitError(ctx context.Context, codes VerificationCodeStore, email string, now time.Time) error {
	recent, err := codes.CountByEmailSince(ctx, email, now.Add(-auth.CodeSendInterval))
	if err != nil {
		return apperr.Internal(err)
	}
	if recent > 0 {
		return errCodeSentRecently
	}
	return errTooManyCodeSends
}

// warnIfManyMails は全体で送ったコードのメールの数を数え、Gmail の上限に近づいていたら警告のログを出す（AUTHN-6）。
// 数えられなくても送信は止めない。
func warnIfManyMails(ctx context.Context, codes VerificationCodeStore, logger *slog.Logger, now time.Time) {
	n, err := codes.CountSince(ctx, now.Add(-auth.MailWarnWindow))
	if err != nil {
		logger.ErrorContext(ctx, "コードのメールの送信数を数えられませんでした", "error", err)
		return
	}
	if n >= auth.MailWarnThreshold {
		// 本番のアラート（infra/setup.sh）はこの文言との完全一致で検知する。変えるときは両方を直す
		logger.WarnContext(ctx, "コードのメールの送信数が1日の上限に近づいています",
			"count", n, "threshold", auth.MailWarnThreshold, "window_hours", int(auth.MailWarnWindow.Hours()))
	}
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
