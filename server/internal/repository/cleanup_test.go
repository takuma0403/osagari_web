package repository_test

import (
	"context"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/repository"
	"github.com/takuma0403/osagari_web/server/internal/testutil/testdb"
)

// 承認期限を過ぎた購入希望だけを、期限の古い順に返す（DEAL-1, JOBS-1）。
func TestTransactionRepository_ListExpiredRequests(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	f := newDealFixture(t, gdb)
	txs := repository.NewTransactionRepository(gdb)
	early := f.newRequest(f.buyer2)
	early.ApprovalDeadline = t0.Add(47 * time.Hour)
	late := f.newRequest(f.buyer)
	rejected := f.newRequest(f.buyer) // 同じ購入者の却下済みの購入希望
	rejected.Status = domain.TransactionStatusRejected
	for _, tr := range []*domain.Transaction{late, early, rejected} {
		if err := txs.Create(ctx, tr); err != nil {
			t.Fatal(err)
		}
	}

	ids := func(now time.Time) []string {
		t.Helper()
		got, err := txs.ListExpiredRequests(ctx, now)
		if err != nil {
			t.Fatal(err)
		}
		out := make([]string, len(got))
		for i, tr := range got {
			out[i] = tr.ID
		}
		return out
	}
	if got := ids(t0.Add(47*time.Hour - time.Second)); len(got) != 0 {
		t.Errorf("期限の前: got = %v", got)
	}
	if got := ids(t0.Add(48 * time.Hour)); len(got) != 2 || got[0] != early.ID || got[1] != late.ID {
		t.Errorf("期限ちょうど: got = %v, want [%s %s]（却下済みは含めない）", got, early.ID, late.ID)
	}
}

// 期限切れのコード・トークンを消す（JOBS-1）。期限ちょうどは消し、期限内は残す。
func TestDeleteExpired(t *testing.T) {
	ctx := context.Background()
	exp := t0.Add(time.Hour)

	t.Run("受け渡しトークン", func(t *testing.T) {
		gdb := testdb.New(t)
		f := newDealFixture(t, gdb)
		tr := f.newRequest(f.buyer)
		if err := repository.NewTransactionRepository(gdb).Create(ctx, tr); err != nil {
			t.Fatal(err)
		}
		toks := repository.NewHandoverTokenRepository(gdb)
		for i, e := range []time.Time{exp, exp.Add(time.Second)} {
			tok := &domain.HandoverToken{
				TransactionID: tr.ID, Kind: domain.HandoverTokenKindLink, TokenHash: ptr([]string{"a", "b"}[i]),
				ExpiresAt: e, CreatedAt: t0, UpdatedAt: t0,
			}
			if err := toks.Create(ctx, tok); err != nil {
				t.Fatal(err)
			}
		}
		wantDeleted(t, gdb, "handover_tokens", 1, func() (int, error) { return toks.DeleteExpired(ctx, exp) })
	})

	t.Run("再確認のチャレンジ", func(t *testing.T) {
		gdb := testdb.New(t)
		u := newUser("", email, "26238001")
		if err := repository.NewUserRepository(gdb).Create(ctx, u); err != nil {
			t.Fatal(err)
		}
		chs := repository.NewLoginChallengeRepository(gdb)
		for i, e := range []time.Time{exp, exp.Add(time.Second)} {
			c := &domain.LoginChallenge{
				UserID: u.ID, TokenHash: []string{"a", "b"}[i], Reason: domain.LoginRecheckReasonNewDevice,
				ExpiresAt: e, CreatedAt: t0, UpdatedAt: t0,
			}
			if err := chs.Create(ctx, c); err != nil {
				t.Fatal(err)
			}
		}
		wantDeleted(t, gdb, "login_challenges", 1, func() (int, error) { return chs.DeleteExpired(ctx, exp) })
	})

	t.Run("パスワード設定のトークン", func(t *testing.T) {
		gdb := testdb.New(t)
		toks := repository.NewPasswordSetupTokenRepository(gdb)
		for i, e := range []time.Time{exp, exp.Add(time.Second)} {
			tok := &domain.PasswordSetupToken{
				Email: email, Purpose: domain.PasswordSetupPurposeSignup, TokenHash: []string{"a", "b"}[i],
				ExpiresAt: e, CreatedAt: t0, UpdatedAt: t0,
			}
			if err := toks.Create(ctx, tok); err != nil {
				t.Fatal(err)
			}
		}
		wantDeleted(t, gdb, "password_setup_tokens", 1, func() (int, error) { return toks.DeleteExpired(ctx, exp) })
	})

	t.Run("認証コードは作ってから一定の間は消さない", func(t *testing.T) {
		gdb := testdb.New(t)
		codes := repository.NewVerificationCodeRepository(gdb)
		for _, c := range []struct{ created, expires time.Time }{
			{t0, exp}, // 期限切れ・作成から24時間 → 消す
			{t0.Add(time.Second), exp.Add(time.Second)}, // 期限切れ・作成から24時間未満 → 残す
			{t0, t0.Add(25 * time.Hour)},                // 期限内 → 残す
		} {
			vc := &domain.VerificationCode{
				Email: email, Purpose: domain.VerificationPurposeSignup, CodeHash: "hash",
				ExpiresAt: c.expires, CreatedAt: c.created, UpdatedAt: c.created,
			}
			if err := codes.Create(ctx, vc); err != nil {
				t.Fatal(err)
			}
		}
		now := t0.Add(24 * time.Hour)
		wantDeleted(t, gdb, "verification_codes", 2, func() (int, error) {
			return codes.DeleteExpired(ctx, now, now.Add(-24*time.Hour))
		})
	})
}

// wantDeleted は del が1件を消し、table に remain 件が残ることを確かめる。
func wantDeleted(t *testing.T, gdb *gorm.DB, table string, remain int64, del func() (int, error)) {
	t.Helper()
	if n, err := del(); err != nil || n != 1 {
		t.Fatalf("消した件数: n=%d err=%v want 1", n, err)
	}
	var got int64
	if err := gdb.Table(table).Count(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got != remain {
		t.Errorf("%s の残り = %d, want %d", table, got, remain)
	}
}
