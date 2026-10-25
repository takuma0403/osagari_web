package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/repository"
	"github.com/takuma0403/osagari_web/server/internal/testutil/testdb"
)

// 受け渡し完了の条件付きUPDATE：購入者の取引中の取引だけを完了にする（DEAL-3, DB）。
func TestTransactionRepository_Complete(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	f := newDealFixture(t, gdb)
	txs := repository.NewTransactionRepository(gdb)
	req := f.newRequest(f.buyer)
	if err := txs.Create(ctx, req); err != nil {
		t.Fatal(err)
	}
	t1 := t0.Add(time.Hour)

	if err := txs.Complete(ctx, req.ID, f.buyer.ID, domain.HandoverMethodQR, t1); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("承認前: err = %v, want ErrConflict", err)
	}
	if err := txs.Approve(ctx, req.ID, f.seller.ID, 500, 75, 1425, t1); err != nil {
		t.Fatal(err)
	}
	if err := txs.Complete(ctx, req.ID, f.seller.ID, domain.HandoverMethodQR, t1); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("出品者: err = %v, want ErrConflict", err)
	}
	if err := txs.Complete(ctx, req.ID, f.buyer.ID, domain.HandoverMethodLink, t1); err != nil {
		t.Fatalf("完了: %v", err)
	}
	got, err := txs.FindByID(ctx, req.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.TransactionStatusCompleted || !got.CompletedAt.Equal(t1) || *got.HandoverMethod != domain.HandoverMethodLink {
		t.Errorf("完了後: got = %+v", got)
	}
	if err := txs.Complete(ctx, req.ID, f.buyer.ID, domain.HandoverMethodQR, t1); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("二重完了: err = %v, want ErrConflict", err)
	}
}

// 受け渡しトークン：検索・無効化・一度だけの使用（DEAL-3）。
func TestHandoverTokenRepository(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	f := newDealFixture(t, gdb)
	txs := repository.NewTransactionRepository(gdb)
	toks := repository.NewHandoverTokenRepository(gdb)
	req := f.newRequest(f.buyer)
	if err := txs.Create(ctx, req); err != nil {
		t.Fatal(err)
	}
	newToken := func(kind domain.HandoverTokenKind, tokenHash, codeHash *string) *domain.HandoverToken {
		t.Helper()
		tok := &domain.HandoverToken{
			TransactionID: req.ID, Kind: kind, TokenHash: tokenHash, CodeHash: codeHash,
			ExpiresAt: t0.Add(kind.TTL()), CreatedAt: t0, UpdatedAt: t0,
		}
		if err := toks.Create(ctx, tok); err != nil {
			t.Fatal(err)
		}
		return tok
	}
	qr := newToken(domain.HandoverTokenKindQR, ptr("qr-hash"), ptr("qr-code-hash"))
	link := newToken(domain.HandoverTokenKindLink, ptr("link-hash"), nil)
	code := newToken(domain.HandoverTokenKindCode, nil, ptr("code-hash"))

	t.Run("種類と列の組み合わせの制約", func(t *testing.T) {
		bad := &domain.HandoverToken{TransactionID: req.ID, Kind: domain.HandoverTokenKindLink, CodeHash: ptr("x"), ExpiresAt: t0, CreatedAt: t0, UpdatedAt: t0}
		if err := toks.Create(ctx, bad); err == nil {
			t.Error("link にコードだけ: err = nil")
		}
	})

	got, err := toks.FindActiveByTokenHash(ctx, req.ID, "qr-hash", t0)
	if err != nil || got.ID != qr.ID || got.Kind != domain.HandoverTokenKindQR || *got.CodeHash != "qr-code-hash" {
		t.Errorf("トークンで検索: got = %+v, err = %v", got, err)
	}
	if _, err := toks.FindActiveByTokenHash(ctx, "other", "qr-hash", t0); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("ほかの取引: err = %v, want ErrNotFound", err)
	}
	if _, err := toks.FindActiveByTokenHash(ctx, req.ID, "qr-hash", qr.ExpiresAt); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("期限ちょうど: err = %v, want ErrNotFound", err)
	}
	withCode, err := toks.ListActiveWithCode(ctx, req.ID, t0)
	if err != nil || len(withCode) != 2 {
		t.Fatalf("コードを持つトークン: got = %d, err = %v", len(withCode), err)
	}

	// 同じ種類の古いトークンだけを無効にする
	qr2 := newToken(domain.HandoverTokenKindQR, ptr("qr2-hash"), ptr("qr2-code-hash"))
	if err := toks.InvalidateOthers(ctx, req.ID, domain.HandoverTokenKindQR, qr2.ID, t0); err != nil {
		t.Fatal(err)
	}
	if _, err := toks.FindActiveByTokenHash(ctx, req.ID, "qr-hash", t0); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("無効にした QR: err = %v, want ErrNotFound", err)
	}
	if err := toks.Consume(ctx, qr.ID, t0); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("無効にした QR の使用: err = %v, want ErrConflict", err)
	}
	for _, h := range []string{"qr2-hash", "link-hash"} {
		if _, err := toks.FindActiveByTokenHash(ctx, req.ID, h, t0); err != nil {
			t.Errorf("%s: %v", h, err)
		}
	}

	// 一度だけ使える
	if err := toks.Consume(ctx, link.ID, link.ExpiresAt); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("期限ちょうどの使用: err = %v, want ErrConflict", err)
	}
	if err := toks.Consume(ctx, code.ID, t0); err != nil {
		t.Fatalf("使用: %v", err)
	}
	if err := toks.Consume(ctx, code.ID, t0); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("二重使用: err = %v, want ErrConflict", err)
	}
	withCode, err = toks.ListActiveWithCode(ctx, req.ID, t0)
	if err != nil || len(withCode) != 1 || withCode[0].ID != qr2.ID {
		t.Errorf("使用・無効の後のコードを持つトークン: got = %+v, err = %v", withCode, err)
	}
}
