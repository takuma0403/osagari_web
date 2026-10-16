package service_test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/auth"
	"github.com/takuma0403/osagari_web/server/internal/repository"
	"github.com/takuma0403/osagari_web/server/internal/service"
)

// コード送信の回数制限（AUTHN-6, ADR 0011）。同じアドレスへの送信は用途をまとめて1分に1回・1時間に5回まで。
func Testコード送信の回数制限(t *testing.T) {
	ctx := context.Background()

	t.Run("1分以内の再送は 429 で、1分たてば送れる", func(t *testing.T) {
		f := newFixture(t)
		if err := f.svc.SendCode(ctx, email); err != nil {
			t.Fatal(err)
		}
		sent := len(f.mailer.Messages())
		f.clock.Advance(auth.CodeSendInterval - time.Second)
		err := f.svc.SendCode(ctx, email)
		wantCode(t, err, apperr.CodeTooManyRequests)
		if !strings.Contains(err.Error(), "1分ほど") {
			t.Errorf("文言: %v", err)
		}
		if n := len(f.mailer.Messages()) - sent; n != 0 {
			t.Errorf("断ったのに%d通送った", n)
		}
		f.clock.Advance(time.Second)
		if err := f.svc.SendCode(ctx, email); err != nil {
			t.Errorf("1分たっても送れない: %v", err)
		}
	})

	t.Run("1時間に6回目は 429 で、それまでのコードは使え、最初の送信から1時間たてば送れる", func(t *testing.T) {
		f := newFixture(t)
		for i := range auth.MaxCodeSends {
			if err := f.svc.SendCode(ctx, email); err != nil {
				t.Fatalf("%d回目: %v", i+1, err)
			}
			f.clock.Advance(10 * time.Minute)
		}
		last := f.lastCode(t)
		// 5回目から10分後（最初の送信から50分後）
		err := f.svc.SendCode(ctx, email)
		wantCode(t, err, apperr.CodeTooManyRequests)
		if !strings.Contains(err.Error(), "1時間ほど") {
			t.Errorf("文言: %v", err)
		}
		f.clock.Advance(9*time.Minute + 59*time.Second)
		wantCode(t, f.svc.SendCode(ctx, email), apperr.CodeTooManyRequests)
		// 断っても、最後に送ったコードは無効にならない（このときまだ有効期限内）
		f.clock.Set(t0.Add(40*time.Minute + 5*time.Minute))
		if _, err := f.svc.VerifyCode(ctx, email, last); err != nil {
			t.Errorf("断った後に、それまでのコードが使えない: %v", err)
		}
		f.clock.Set(t0.Add(auth.CodeSendWindow))
		if err := f.svc.SendCode(ctx, email); err != nil {
			t.Errorf("最初の送信から1時間たっても送れない: %v", err)
		}
	})

	t.Run("登録済みのアドレスでも未登録と同じように 429 になる（AUTHN-2）", func(t *testing.T) {
		f := newFixture(t)
		f.signup(t, email, "")
		f.clock.Advance(auth.CodeSendInterval)
		const other = "26238002@edu.cc.saga-u.ac.jp"
		for _, addr := range []string{email, other} {
			if err := f.svc.SendCode(ctx, addr); err != nil {
				t.Fatal(err)
			}
		}
		errRegistered := f.svc.SendCode(ctx, email)
		errNew := f.svc.SendCode(ctx, other)
		wantCode(t, errRegistered, apperr.CodeTooManyRequests)
		if errRegistered.Error() != errNew.Error() {
			t.Errorf("応答が違う: 登録済み=%v 未登録=%v", errRegistered, errNew)
		}
	})

	t.Run("用途が違っても同じアドレスへの送信として数える", func(t *testing.T) {
		f := newResetFixture(t)
		if err := f.reset.SendCode(ctx, email); err != nil {
			t.Fatal(err)
		}
		wantCode(t, f.svc.SendCode(ctx, email), apperr.CodeTooManyRequests)
		_, err := f.login.Login(ctx, email, password, "new-device")
		wantCode(t, err, apperr.CodeTooManyRequests)
	})

	t.Run("ログインの再確認のコードの再送も数える", func(t *testing.T) {
		f := newLoginFixture(t)
		c := f.challenge(t, "new-device")
		wantCode(t, f.login.ResendRecheckCode(ctx, c.Token), apperr.CodeTooManyRequests)
		f.clock.Advance(auth.CodeSendInterval)
		if err := f.login.ResendRecheckCode(ctx, c.Token); err != nil {
			t.Errorf("1分たっても再送できない: %v", err)
		}
	})
}

// 全体の送信数が Gmail の上限に近づいたら警告のログを出す（AUTHN-6）。送信は止めない。
func Test全体の送信数の警告(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	var buf bytes.Buffer
	svc := service.NewSignupService(service.SignupDeps{
		Tx:       repository.NewTransactor(f.gdb),
		Users:    repository.NewUserRepository(f.gdb),
		Codes:    repository.NewVerificationCodeRepository(f.gdb),
		Tokens:   repository.NewPasswordSetupTokenRepository(f.gdb),
		Sessions: repository.NewSessionRepository(f.gdb),
		Devices:  repository.NewTrustedDeviceRepository(f.gdb),
		Mailer:   f.mailer,
		Clock:    f.clock,
		Logger:   slog.New(slog.NewTextHandler(&buf, nil)),
	})
	// 24時間より前の送信は数えない
	insertCodes(t, f, 10, t0.Add(-auth.MailWarnWindow))
	insertCodes(t, f, auth.MailWarnThreshold-2, t0.Add(-auth.MailWarnWindow+time.Second))

	if err := svc.SendCode(ctx, "26238101@edu.cc.saga-u.ac.jp"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "level=WARN") {
		t.Errorf("上限の手前で警告した: %s", buf.String())
	}
	if err := svc.SendCode(ctx, "26238102@edu.cc.saga-u.ac.jp"); err != nil {
		t.Errorf("警告の数に達しても送信は止めない: %v", err)
	}
	if !strings.Contains(buf.String(), "level=WARN") || !strings.Contains(buf.String(), fmt.Sprintf("count=%d", auth.MailWarnThreshold)) {
		t.Errorf("警告が出ていない: %s", buf.String())
	}
	if strings.Contains(buf.String(), "@") {
		t.Errorf("ログにメールアドレスが出ている: %s", buf.String())
	}
}

// insertCodes は別々のアドレスに at に送ったコードを n 件保存する。
func insertCodes(t *testing.T, f *fixture, n int, at time.Time) {
	t.Helper()
	ts := at.UTC().Format("2006-01-02T15:04:05.000000Z")
	for i := range n {
		err := f.gdb.Exec(`INSERT INTO verification_codes (id, email, purpose, code_hash, expires_at, created_at, updated_at)
			VALUES (?, ?, 'signup', 'hash', ?, ?, ?)`, fmt.Sprintf("%s-%d", ts, i), fmt.Sprintf("bulk%d@example.com", i), ts, ts, ts).Error
		if err != nil {
			t.Fatal(err)
		}
	}
}
