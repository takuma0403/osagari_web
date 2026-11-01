package service

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/auth"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/repository"
)

// 受け渡しトークン（QR・完了リンク・完了コード）の発行・照合と受け渡し完了（DEAL-3, PAY-2, JOBS-2）。

// CodeInvalidHandoverToken は受け渡しトークンが違う・期限切れ・使用済みのときのエラーコード。
const CodeInvalidHandoverToken apperr.Code = "invalid_handover_token"

var (
	errHandoverKind    = apperr.BadRequest("受け渡しトークンの種類が正しくありません")
	errHandoverQRToken = apperr.New(http.StatusBadRequest, CodeInvalidHandoverToken,
		"このQRコードは使えません。有効期限が切れたか、すでに使われています。出品者にQRを表示し直してもらってください")
	errHandoverLinkToken = apperr.New(http.StatusBadRequest, CodeInvalidHandoverToken,
		"この完了リンクは使えません。有効期限（24時間）が切れたか、すでに使われています。出品者に新しいリンクかコードを発行してもらってください")
	errHandoverCode = apperr.New(http.StatusBadRequest, CodeInvalidHandoverToken,
		"コードが違うか、有効期限が切れています。出品者に新しいコードを発行してもらってください")
)

// IssuedHandoverToken は発行した受け渡しトークン。平文の値はこのときだけ返す。
// qr は Token と Code、link は Token、code は Code だけが入る。
type IssuedHandoverToken struct {
	Kind      domain.HandoverTokenKind
	Token     *string
	Code      *string
	ExpiresAt time.Time
}

// HandoverCredential は購入者が読み取った・開いた・入力したトークン。Kind は入力の種類（QR の下の確認コードを入力したときは code）。
type HandoverCredential struct {
	Kind   domain.HandoverTokenKind
	Secret string
}

// IssueHandoverToken は出品者が受け渡しトークンを発行する（DEAL-3）。同じ取引・同じ種類の未使用のトークンは使えなくする。
// 出品者でなければ 404、取引中でなければ 409。
func (s *DealService) IssueHandoverToken(ctx context.Context, u *domain.User, id string, kind domain.HandoverTokenKind) (*IssuedHandoverToken, error) {
	if !kind.Valid() {
		return nil, errHandoverKind
	}
	tr, err := s.findAsSeller(ctx, u, id)
	if err != nil {
		return nil, err
	}
	if tr.Status != domain.TransactionStatusInDeal {
		return nil, errTransactionConflict
	}
	now := s.d.Clock.Now()
	out := &IssuedHandoverToken{Kind: kind, ExpiresAt: now.Add(kind.TTL())}
	t := &domain.HandoverToken{
		TransactionID: tr.ID,
		Kind:          kind,
		ExpiresAt:     out.ExpiresAt,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if kind.HasToken() {
		token, err := auth.GenerateToken()
		if err != nil {
			return nil, apperr.Internal(err)
		}
		h := auth.HashToken(token)
		out.Token, t.TokenHash = &token, &h
	}
	if kind.HasCode() {
		code, err := auth.GenerateCode()
		if err != nil {
			return nil, apperr.Internal(err)
		}
		h, err := auth.HashCode(code)
		if err != nil {
			return nil, apperr.Internal(err)
		}
		out.Code, t.CodeHash = &code, &h
	}
	err = s.d.Tx.Do(ctx, func(ctx context.Context) error {
		if err := s.d.HandoverTokens.Create(ctx, t); err != nil {
			return err
		}
		return s.d.HandoverTokens.InvalidateOthers(ctx, tr.ID, kind, t.ID, now)
	})
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return out, nil
}

// VerifyHandover は購入者の入力したトークンが使えるかを確かめる。使用済みにはしない（DEAL-3。「商品を受け取りましたか？」の前の照合）。
// 購入者でなければ 404、取引中でなければ 409、使えなければ 400 invalid_handover_token。
func (s *DealService) VerifyHandover(ctx context.Context, u *domain.User, id string, cred HandoverCredential) error {
	tr, err := s.findInDealAsBuyer(ctx, u, id)
	if err != nil {
		return err
	}
	_, err = s.findHandoverToken(ctx, tr.ID, cred, s.d.Clock.Now())
	return err
}

// Complete は購入者の操作で受け渡しを完了する（DEAL-3, PAY-2, JOBS-2）。
// トークンを使用済みにし、取引と商品を完了にして、取引成立時に記録した出品者の売上を確定する。完了方法は入力の種類で記録する（ADR 0024）。
// 購入者でなければ 404（出品者の操作では完了しない）、取引中でない・同時に完了されたなら 409、使えなければ 400 invalid_handover_token。
func (s *DealService) Complete(ctx context.Context, u *domain.User, id string, cred HandoverCredential) (*TransactionView, error) {
	tr, err := s.findInDealAsBuyer(ctx, u, id)
	if err != nil {
		return nil, err
	}
	now := s.d.Clock.Now()
	t, err := s.findHandoverToken(ctx, tr.ID, cred, now)
	if err != nil {
		return nil, err
	}
	err = s.d.Tx.Do(ctx, func(ctx context.Context) error {
		// DEAL-3: トークンは一度使ったら無効にする。同じトークンで同時に完了されたら、後の方はここで競合になる
		if err := s.d.HandoverTokens.Consume(ctx, t.ID, now); err != nil {
			return err
		}
		if err := s.d.Transactions.Complete(ctx, tr.ID, u.ID, domain.HandoverMethodOf(cred.Kind), now); err != nil {
			return err
		}
		err := s.d.Items.UpdateDealStatus(ctx, tr.ItemID, tr.SellerID, []domain.ItemStatus{domain.ItemStatusInDeal}, domain.ItemStatusCompleted, now)
		// 商品が取引中でなければ（管理者の停止など）そのままにする。受け渡しの完了は止めない
		if err != nil && !errors.Is(err, repository.ErrConflict) {
			return err
		}
		return s.d.Payments.Settle(ctx, tr)
	})
	if err := dealError(err); err != nil {
		return nil, err
	}
	return s.view(ctx, u, tr.ID)
}

// findInDealAsBuyer は本人が購入者の取引を返す。ないか、購入者でなければ 404（出品者にも存在を区別させない）、取引中でなければ 409。
func (s *DealService) findInDealAsBuyer(ctx context.Context, u *domain.User, id string) (*domain.Transaction, error) {
	tr, err := s.findAs(ctx, u, id, domain.TransactionRoleBuyer)
	if err != nil {
		return nil, err
	}
	if tr.Status != domain.TransactionStatusInDeal {
		return nil, errTransactionConflict
	}
	return tr, nil
}

// findHandoverToken は入力に合う、取引の未使用・期限内のトークンを返す。なければ入力の種類に合わせた 400 invalid_handover_token。
// qr・link はトークンの SHA-256 で探し、種類も一致させる。code は6桁のコードを持つトークン（qr の確認コードと完了コード）を argon2id で照合する。
func (s *DealService) findHandoverToken(ctx context.Context, transactionID string, cred HandoverCredential, now time.Time) (*domain.HandoverToken, error) {
	switch cred.Kind {
	case domain.HandoverTokenKindQR, domain.HandoverTokenKindLink:
		invalid := errHandoverQRToken
		if cred.Kind == domain.HandoverTokenKindLink {
			invalid = errHandoverLinkToken
		}
		if cred.Secret == "" {
			return nil, invalid
		}
		t, err := s.d.HandoverTokens.FindActiveByTokenHash(ctx, transactionID, auth.HashToken(cred.Secret), now)
		if errors.Is(err, repository.ErrNotFound) || (err == nil && t.Kind != cred.Kind) {
			return nil, invalid
		}
		if err != nil {
			return nil, apperr.Internal(err)
		}
		return t, nil
	case domain.HandoverTokenKindCode:
		if !sixDigits.MatchString(cred.Secret) {
			return nil, errHandoverCode
		}
		ts, err := s.d.HandoverTokens.ListActiveWithCode(ctx, transactionID, now)
		if err != nil {
			return nil, apperr.Internal(err)
		}
		for _, t := range ts {
			ok, err := auth.VerifyCode(*t.CodeHash, cred.Secret)
			if err != nil {
				return nil, apperr.Internal(err)
			}
			if ok {
				return t, nil
			}
		}
		return nil, errHandoverCode
	default:
		return nil, errHandoverKind
	}
}
