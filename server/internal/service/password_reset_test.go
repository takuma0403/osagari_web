package service_test

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/auth"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/repository"
	"github.com/takuma0403/osagari_web/server/internal/service"
)

const newPassword = "brand new password"

type resetFixture struct {
	*loginFixture
	reset *service.PasswordResetService
}

// newResetFixture は newLoginFixture のユーザーに加えて、パスワード再設定の service を用意する。
func newResetFixture(t *testing.T) *resetFixture {
	t.Helper()
	f := &resetFixture{loginFixture: newLoginFixture(t)}
	f.reset = service.NewPasswordResetService(service.PasswordResetDeps{
		Tx:       repository.NewTransactor(f.gdb),
		Users:    repository.NewUserRepository(f.gdb),
		Codes:    repository.NewVerificationCodeRepository(f.gdb),
		Tokens:   repository.NewPasswordSetupTokenRepository(f.gdb),
		Sessions: repository.NewSessionRepository(f.gdb),
		Devices:  repository.NewTrustedDeviceRepository(f.gdb),
		Mailer:   f.mailer,
		Clock:    f.clock,
	})
	return f
}

// loggedIn は登録済みの端末からログインしてセッションのトークンを返す（再確認なし）。
func (f *resetFixture) loggedIn(t *testing.T) string {
	t.Helper()
	res, err := f.login.Login(context.Background(), email, password, knownDevice)
	if err != nil || res.Session == nil {
		t.Fatalf("ログインできない: res=%+v err=%v", res, err)
	}
	return res.Session.Token
}

// verified はコードの確認まで済ませて一時トークンを返す。
func (f *resetFixture) verified(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	if err := f.reset.SendCode(ctx, email); err != nil {
		t.Fatal(err)
	}
	token, err := f.reset.VerifyCode(ctx, email, f.lastCode(t))
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func TestPasswordReset_SendCode(t *testing.T) {
	ctx := context.Background()

	t.Run("形式が違うアドレスには送らない", func(t *testing.T) {
		f := newResetFixture(t)
		sent := len(f.mailer.Messages())
		for _, e := range []string{"", "2623800@edu.cc.saga-u.ac.jp", "26238001@gmail.com"} {
			wantCode(t, f.reset.SendCode(ctx, e), service.CodeInvalidEmail)
		}
		if n := len(f.mailer.Messages()) - sent; n != 0 {
			t.Errorf("%d通送られた", n)
		}
	})

	t.Run("登録済み・未登録・削除済みで応答と処理が同じで、本文だけが違う（AUTHN-2）", func(t *testing.T) {
		f := newResetFixture(t)
		const deleted = "26238003@edu.cc.saga-u.ac.jp"
		f.signup(t, deleted, "")
		if err := f.gdb.Exec("UPDATE users SET status = ? WHERE email = ?", string(domain.UserStatusDeleted), deleted).Error; err != nil {
			t.Fatal(err)
		}
		f.clock.Advance(auth.CodeSendInterval) // 登録のときのコード送信から間隔をあける（AUTHN-6）
		before := f.count(t, "verification_codes")
		sent := len(f.mailer.Messages())

		type result struct {
			err  error
			body string
			to   string
			subj string
		}
		send := func(addr string) result {
			err := f.reset.SendCode(ctx, addr)
			m, _ := f.mailer.Last()
			return result{err, m.Body, m.To, m.Subject}
		}
		registered := send(email)
		unregistered := send("26238002@edu.cc.saga-u.ac.jp")
		del := send(deleted)

		for _, r := range []result{registered, unregistered, del} {
			if r.err != nil {
				t.Fatalf("応答が違う: %v", r.err)
			}
			if r.subj != registered.subj {
				t.Errorf("件名が違う: %q / %q", r.subj, registered.subj)
			}
		}
		if got := len(f.mailer.Messages()) - sent; got != 3 {
			t.Errorf("送ったメール=%d通 want=3（1通ずつ）", got)
		}
		if got := f.count(t, "verification_codes") - before; got != 3 {
			t.Errorf("保存したコード=%d件 want=3（未登録でもコードを保存する）", got)
		}
		if registered.to != email || !codeInMail.MatchString(registered.body) {
			t.Errorf("登録済みにコードが届いていない: %+v", registered)
		}
		for _, r := range []result{unregistered, del} {
			if codeInMail.MatchString(r.body) || !strings.Contains(r.body, "登録されていません") {
				t.Errorf("未登録の案内になっていない: %s", r.body)
			}
		}
	})

	t.Run("停止中・失効のアカウントにもコードを送る", func(t *testing.T) {
		for _, st := range []domain.UserStatus{domain.UserStatusSuspended, domain.UserStatusExpired} {
			f := newResetFixture(t)
			f.setStatus(t, st)
			if err := f.reset.SendCode(ctx, email); err != nil {
				t.Fatal(err)
			}
			if f.lastCode(t) == "" {
				t.Errorf("%s: コードが書かれていない", st)
			}
		}
	})

	t.Run("メールを送れなければ 500", func(t *testing.T) {
		f := newResetFixture(t)
		f.mailer.Err = context.DeadlineExceeded
		wantCode(t, f.reset.SendCode(ctx, email), apperr.CodeInternal)
	})
}

func TestPasswordReset_VerifyCode(t *testing.T) {
	ctx := context.Background()

	t.Run("コードが一致したら一時トークンを返す", func(t *testing.T) {
		f := newResetFixture(t)
		if f.verified(t) == "" {
			t.Error("トークンが空")
		}
	})

	cases := []struct {
		name string
		run  func(t *testing.T, f *resetFixture, code string) error
		want apperr.Code
	}{
		{"コードが違う", func(t *testing.T, f *resetFixture, code string) error {
			_, err := f.reset.VerifyCode(ctx, email, wrongCode(code))
			return err
		}, service.CodeInvalidVerificationCode},
		{"10分たったコード", func(t *testing.T, f *resetFixture, code string) error {
			f.clock.Advance(auth.CodeTTL)
			_, err := f.reset.VerifyCode(ctx, email, code)
			return err
		}, service.CodeVerificationCodeExpired},
		{"新規登録のコードとは別", func(t *testing.T, f *resetFixture, code string) error {
			_, err := f.svc.VerifyCode(ctx, email, code)
			return err
		}, service.CodeVerificationCodeExpired},
		{"形式が違うアドレス", func(t *testing.T, f *resetFixture, code string) error {
			_, err := f.reset.VerifyCode(ctx, "26238001@gmail.com", code)
			return err
		}, service.CodeInvalidEmail},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newResetFixture(t)
			if err := f.reset.SendCode(ctx, email); err != nil {
				t.Fatal(err)
			}
			wantCode(t, tc.run(t, f, f.lastCode(t)), tc.want)
		})
	}

	t.Run("未登録のアドレスに保存したコードは当てられない", func(t *testing.T) {
		f := newResetFixture(t)
		const other = "26238002@edu.cc.saga-u.ac.jp"
		if err := f.reset.SendCode(ctx, other); err != nil {
			t.Fatal(err)
		}
		_, err := f.reset.VerifyCode(ctx, other, "123456")
		wantCode(t, err, service.CodeInvalidVerificationCode)
	})
}

func TestPasswordReset_Complete(t *testing.T) {
	ctx := context.Background()

	t.Run("パスワードを変え、既存のセッションをすべて無効にして、この端末でログインする（AUTHN-4）", func(t *testing.T) {
		f := newResetFixture(t)
		old1, old2 := f.loggedIn(t), f.loggedIn(t)
		token := f.verified(t)
		f.clock.Advance(time.Minute)
		now := t0.Add(time.Minute)

		out, err := f.reset.Complete(ctx, token, newPassword, "device-2")
		if err != nil {
			t.Fatal(err)
		}
		if out.User.ID != f.user.ID || !out.User.CodeVerifiedAt.Equal(now) {
			t.Errorf("user=%+v", out.User)
		}
		if out.Session.Token == "" || !out.Session.ExpiresAt.Equal(now.Add(auth.SessionTTL)) {
			t.Errorf("session=%+v", out.Session)
		}
		for _, old := range []string{old1, old2} {
			_, err := f.sessions.Authenticate(ctx, old)
			wantCode(t, err, apperr.CodeUnauthorized)
		}
		if u, err := f.sessions.Authenticate(ctx, out.Session.Token); err != nil || u.ID != f.user.ID {
			t.Errorf("新しいセッションが使えない: u=%+v err=%v", u, err)
		}

		// 新しいパスワードでログインでき、古いパスワードは使えない。この端末は登録済みなので再確認は要らない
		res, err := f.login.Login(ctx, email, newPassword, "device-2")
		if err != nil || res.Session == nil {
			t.Errorf("新しいパスワードでログインできない: res=%+v err=%v", res, err)
		}
		_, err = f.login.Login(ctx, email, password, knownDevice)
		wantCode(t, err, service.CodeInvalidCredentials)
	})

	t.Run("最後の確認から30日以上でも、再設定でコードの確認日時が更新される", func(t *testing.T) {
		f := newResetFixture(t)
		f.clock.Advance(auth.RecheckInterval + time.Hour)
		if _, err := f.reset.Complete(ctx, f.verified(t), newPassword, ""); err != nil {
			t.Fatal(err)
		}
		res, err := f.login.Login(ctx, email, newPassword, knownDevice)
		if err != nil || res.Session == nil {
			t.Errorf("再確認なしでログインできない: res=%+v err=%v", res, err)
		}
	})

	cases := []struct {
		name string
		run  func(t *testing.T, f *resetFixture, token string) error
		want apperr.Code
	}{
		{"パスワードが9文字ならトークンを消費しない", func(t *testing.T, f *resetFixture, token string) error {
			_, err := f.reset.Complete(ctx, token, "123456789", "")
			if _, err2 := f.reset.Complete(ctx, token, newPassword, ""); err2 != nil {
				t.Errorf("入力し直せない: %v", err2)
			}
			return err
		}, service.CodeInvalidPasswordLength},
		{"パスワードが129文字", func(t *testing.T, f *resetFixture, token string) error {
			_, err := f.reset.Complete(ctx, token, strings.Repeat("a", 129), "")
			return err
		}, service.CodeInvalidPasswordLength},
		{"15分たったトークン", func(t *testing.T, f *resetFixture, token string) error {
			f.clock.Advance(auth.SetupTokenTTL)
			_, err := f.reset.Complete(ctx, token, newPassword, "")
			return err
		}, service.CodeInvalidSetupToken},
		{"使用済みのトークン", func(t *testing.T, f *resetFixture, token string) error {
			if _, err := f.reset.Complete(ctx, token, newPassword, ""); err != nil {
				t.Fatal(err)
			}
			_, err := f.reset.Complete(ctx, token, newPassword, "")
			return err
		}, service.CodeInvalidSetupToken},
		{"存在しないトークン", func(t *testing.T, f *resetFixture, token string) error {
			_, err := f.reset.Complete(ctx, token+"x", newPassword, "")
			return err
		}, service.CodeInvalidSetupToken},
		{"新規登録用のトークンでは再設定できない", func(t *testing.T, f *resetFixture, _ string) error {
			if err := f.svc.SendCode(ctx, "26238002@edu.cc.saga-u.ac.jp"); err != nil {
				t.Fatal(err)
			}
			v, err := f.svc.VerifyCode(ctx, "26238002@edu.cc.saga-u.ac.jp", f.lastCode(t))
			if err != nil {
				t.Fatal(err)
			}
			_, err = f.reset.Complete(ctx, v.SetupToken, newPassword, "")
			return err
		}, service.CodeInvalidSetupToken},
		{"削除済みのアカウント", func(t *testing.T, f *resetFixture, token string) error {
			f.setStatus(t, domain.UserStatusDeleted)
			_, err := f.reset.Complete(ctx, token, newPassword, "")
			return err
		}, service.CodeInvalidSetupToken},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newResetFixture(t)
			wantCode(t, tc.run(t, f, f.verified(t)), tc.want)
		})
	}

	t.Run("停止中・失効なら理由を返し、パスワード・セッション・トークンはそのまま（ADR 0010）", func(t *testing.T) {
		tests := []struct {
			name  string
			setup func(t *testing.T, f *resetFixture)
			want  apperr.Code
		}{
			{"停止中", func(t *testing.T, f *resetFixture) { f.setStatus(t, domain.UserStatusSuspended) }, service.CodeAccountSuspended},
			{"状態が失効", func(t *testing.T, f *resetFixture) { f.setStatus(t, domain.UserStatusExpired) }, service.CodeAccountExpired},
			{"有効期限の日を過ぎた", func(t *testing.T, f *resetFixture) {
				if err := f.gdb.Exec("UPDATE users SET expires_on = ? WHERE id = ?", "2026-10-13", f.user.ID).Error; err != nil {
					t.Fatal(err)
				}
			}, service.CodeAccountExpired},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				f := newResetFixture(t)
				old := f.loggedIn(t)
				token := f.verified(t)
				tc.setup(t, f)
				_, err := f.reset.Complete(ctx, token, newPassword, "")
				wantCode(t, err, tc.want)

				var hash string
				f.gdb.Raw("SELECT password_hash FROM users WHERE id = ?", f.user.ID).Scan(&hash)
				if ok, _ := auth.VerifyPassword(hash, password); !ok {
					t.Error("パスワードが変わった")
				}
				var revoked int64
				f.gdb.Raw("SELECT COUNT(*) FROM sessions WHERE token_hash = ? AND revoked_at IS NOT NULL", auth.HashToken(old)).Scan(&revoked)
				if revoked != 0 {
					t.Error("既存のセッションが失効した")
				}
				var consumed int64
				f.gdb.Raw("SELECT COUNT(*) FROM password_setup_tokens WHERE token_hash = ? AND consumed_at IS NOT NULL", auth.HashToken(token)).Scan(&consumed)
				if consumed != 0 {
					t.Error("トークンが消費された")
				}
			})
		}
	})

	t.Run("同じトークンで同時に設定しても成功は1回だけ", func(t *testing.T) {
		f := newResetFixture(t)
		token := f.verified(t)
		var ok atomic.Int32
		var wg sync.WaitGroup
		for range 5 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if _, err := f.reset.Complete(ctx, token, newPassword, ""); err == nil {
					ok.Add(1)
				}
			}()
		}
		wg.Wait()
		if ok.Load() != 1 {
			t.Errorf("成功=%d want=1", ok.Load())
		}
	})
}
