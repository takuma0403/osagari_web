package service_test

import (
	"context"
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

const knownDevice = "device-registered-at-signup"

type loginFixture struct {
	*fixture
	login    *service.LoginService
	sessions *service.SessionService
	user     *domain.User
}

// newLoginFixture は t0 に knownDevice から登録したユーザーを用意する（最後のコード確認は t0、knownDevice は登録済み）。
func newLoginFixture(t *testing.T) *loginFixture {
	t.Helper()
	f := &loginFixture{fixture: newFixture(t)}
	f.login = service.NewLoginService(service.LoginDeps{
		Tx:         repository.NewTransactor(f.gdb),
		Users:      repository.NewUserRepository(f.gdb),
		Codes:      repository.NewVerificationCodeRepository(f.gdb),
		Challenges: repository.NewLoginChallengeRepository(f.gdb),
		Failures:   repository.NewLoginFailureRepository(f.gdb),
		Sessions:   repository.NewSessionRepository(f.gdb),
		Devices:    repository.NewTrustedDeviceRepository(f.gdb),
		Mailer:     f.mailer,
		Clock:      f.clock,
	})
	f.sessions = service.NewSessionService(service.SessionDeps{
		Sessions: repository.NewSessionRepository(f.gdb),
		Users:    repository.NewUserRepository(f.gdb),
		Clock:    f.clock,
	})
	f.user = f.signup(t, email, knownDevice).User
	// 準備の登録で送ったコードは、コード送信の回数制限（AUTHN-6）に数えないように消す
	if err := f.gdb.Exec("DELETE FROM verification_codes").Error; err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *loginFixture) setStatus(t *testing.T, status domain.UserStatus) {
	t.Helper()
	if err := f.gdb.Exec("UPDATE users SET status = ? WHERE id = ?", string(status), f.user.ID).Error; err != nil {
		t.Fatal(err)
	}
}

// challenge はログインして再確認のチャレンジを受け取る。
func (f *loginFixture) challenge(t *testing.T, deviceID string) *service.IssuedLoginChallenge {
	t.Helper()
	res, err := f.login.Login(context.Background(), email, password, deviceID)
	if err != nil {
		t.Fatal(err)
	}
	if res.Challenge == nil || res.Session != nil {
		t.Fatalf("再確認になっていない: %+v", res)
	}
	return res.Challenge
}

func TestLogin_再確認の要否(t *testing.T) {
	tests := []struct {
		name       string
		advance    time.Duration
		deviceID   string
		wantReason domain.LoginRecheckReason // 空なら再確認なしでログイン
	}{
		{"登録済みの端末で30日未満ならパスワードだけでログイン", 30*24*time.Hour - time.Second, knownDevice, ""},
		{"初めての端末なら再確認", 0, "another-device", domain.LoginRecheckReasonNewDevice},
		{"端末識別子がなければ初めての端末として再確認", 0, "", domain.LoginRecheckReasonNewDevice},
		{"最後の確認からちょうど30日なら再確認", 30 * 24 * time.Hour, knownDevice, domain.LoginRecheckReasonCodeExpired},
		{"30日以上かつ初めての端末なら code_expired", 31 * 24 * time.Hour, "another-device", domain.LoginRecheckReasonCodeExpired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newLoginFixture(t)
			mailsBefore := len(f.mailer.Messages())
			f.clock.Advance(tt.advance)
			res, err := f.login.Login(context.Background(), " 26238001@EDU.cc.saga-u.ac.jp ", password, tt.deviceID)
			if err != nil {
				t.Fatal(err)
			}
			if res.User == nil || res.User.ID != f.user.ID {
				t.Fatalf("ユーザー: %+v", res.User)
			}
			if tt.wantReason == "" {
				if res.Session == nil || res.Challenge != nil {
					t.Fatalf("セッションが発行されていない: %+v", res)
				}
				if !res.Session.ExpiresAt.Equal(f.clock.Now().Add(auth.SessionTTL)) {
					t.Errorf("セッションの期限=%s", res.Session.ExpiresAt)
				}
				if got, err := f.sessions.Authenticate(context.Background(), res.Session.Token); err != nil || got.ID != f.user.ID {
					t.Errorf("発行したセッションで認証できない: %v", err)
				}
				if n := len(f.mailer.Messages()); n != mailsBefore {
					t.Errorf("再確認なしでメールが送られた: %d通", n-mailsBefore)
				}
				return
			}
			if res.Session != nil || res.Challenge == nil {
				t.Fatalf("再確認になっていない: %+v", res)
			}
			if res.Challenge.Reason != tt.wantReason || res.Challenge.Email != email || res.Challenge.Token == "" {
				t.Errorf("チャレンジ=%+v", res.Challenge)
			}
			m, _ := f.mailer.Last()
			if m.To != email || f.lastCode(t) == "" {
				t.Errorf("コードのメールが送られていない: %+v", m)
			}
			if n := f.count(t, "sessions"); n != 1 {
				t.Errorf("再確認の前にセッションが作られた: sessions=%d", n)
			}
		})
	}
}

func TestLogin_失敗(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, f *loginFixture)
		email    string
		password string
		want     apperr.Code
	}{
		{"パスワード違い", nil, email, password + "x", service.CodeInvalidCredentials},
		{"未登録のアドレス", nil, "26238002@edu.cc.saga-u.ac.jp", password, service.CodeInvalidCredentials},
		{"大学のメールアドレスの形式ではない", nil, "someone@example.com", password, service.CodeInvalidCredentials},
		{"パスワードが短すぎる", nil, email, "short", service.CodeInvalidCredentials},
		{"削除済みは未登録と同じ", func(t *testing.T, f *loginFixture) { f.setStatus(t, domain.UserStatusDeleted) },
			email, password, service.CodeInvalidCredentials},
		{"停止中", func(t *testing.T, f *loginFixture) { f.setStatus(t, domain.UserStatusSuspended) },
			email, password, service.CodeAccountSuspended},
		{"状態が失効", func(t *testing.T, f *loginFixture) { f.setStatus(t, domain.UserStatusExpired) },
			email, password, service.CodeAccountExpired},
		{"状態は有効だが有効期限（2034-03-31）を過ぎた", func(t *testing.T, f *loginFixture) {
			f.clock.Set(time.Date(2034, 3, 31, 15, 0, 0, 0, time.UTC)) // 日本時間 2034-04-01 0:00
		}, email, password, service.CodeAccountExpired},
		{"停止中でもパスワードが違えば理由を出さない", func(t *testing.T, f *loginFixture) { f.setStatus(t, domain.UserStatusSuspended) },
			email, password + "x", service.CodeInvalidCredentials},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newLoginFixture(t)
			if tt.setup != nil {
				tt.setup(t, f)
			}
			mailsBefore := len(f.mailer.Messages())
			_, err := f.login.Login(context.Background(), tt.email, tt.password, knownDevice)
			wantCode(t, err, tt.want)
			if n := len(f.mailer.Messages()); n != mailsBefore {
				t.Errorf("失敗したのにメールが送られた")
			}
			if n := f.count(t, "sessions"); n != 1 {
				t.Errorf("失敗したのにセッションが作られた: sessions=%d", n)
			}
		})
	}
}

func TestVerifyRecheck(t *testing.T) {
	ctx := context.Background()

	t.Run("コードが合えばログインし、確認日時を更新して端末を登録する", func(t *testing.T) {
		f := newLoginFixture(t)
		f.clock.Advance(time.Hour)
		c := f.challenge(t, "new-device")
		code := f.lastCode(t)
		f.clock.Advance(time.Minute)
		out, err := f.login.VerifyRecheck(ctx, c.Token, code)
		if err != nil {
			t.Fatal(err)
		}
		if out.User.ID != f.user.ID || !out.User.CodeVerifiedAt.Equal(f.clock.Now()) {
			t.Errorf("ユーザー=%+v", out.User)
		}
		if _, err := f.sessions.Authenticate(ctx, out.Session.Token); err != nil {
			t.Errorf("発行したセッションで認証できない: %v", err)
		}
		// 次回からこの端末ではパスワードだけでログインできる
		res, err := f.login.Login(ctx, email, password, "new-device")
		if err != nil || res.Session == nil {
			t.Errorf("登録した端末で再確認が求められた: res=%+v err=%v", res, err)
		}
		// 同じチャレンジはもう使えない
		_, err = f.login.VerifyRecheck(ctx, c.Token, code)
		wantCode(t, err, service.CodeInvalidLoginChallenge)
	})

	t.Run("30日たったら、登録済みの端末でも確認後は次の30日までパスワードだけ", func(t *testing.T) {
		f := newLoginFixture(t)
		f.clock.Advance(30 * 24 * time.Hour)
		c := f.challenge(t, knownDevice)
		if _, err := f.login.VerifyRecheck(ctx, c.Token, f.lastCode(t)); err != nil {
			t.Fatal(err)
		}
		f.clock.Advance(29 * 24 * time.Hour)
		if res, err := f.login.Login(ctx, email, password, knownDevice); err != nil || res.Session == nil {
			t.Errorf("確認後29日で再確認が求められた: res=%+v err=%v", res, err)
		}
	})

	t.Run("端末識別子なしでログインした場合は端末を登録しない", func(t *testing.T) {
		f := newLoginFixture(t)
		c := f.challenge(t, "")
		if _, err := f.login.VerifyRecheck(ctx, c.Token, f.lastCode(t)); err != nil {
			t.Fatal(err)
		}
		if n := f.count(t, "trusted_devices"); n != 1 {
			t.Errorf("trusted_devices=%d want=1（登録時の端末だけ）", n)
		}
	})

	t.Run("コード違いは残り回数を返し、5回で使えなくなる", func(t *testing.T) {
		f := newLoginFixture(t)
		c := f.challenge(t, "new-device")
		code := f.lastCode(t)
		for i := 1; i <= auth.MaxCodeFailures; i++ {
			_, err := f.login.VerifyRecheck(ctx, c.Token, wrongCode(code))
			wantCode(t, err, service.CodeInvalidVerificationCode)
		}
		_, err := f.login.VerifyRecheck(ctx, c.Token, code)
		wantCode(t, err, service.CodeVerificationCodeExpired)
	})

	t.Run("コードを再送すると古いコードは使えず、新しいコードで確認できる", func(t *testing.T) {
		f := newLoginFixture(t)
		c := f.challenge(t, "new-device")
		old := f.lastCode(t)
		f.clock.Advance(auth.CodeSendInterval)
		if err := f.login.ResendRecheckCode(ctx, c.Token); err != nil {
			t.Fatal(err)
		}
		fresh := f.lastCode(t)
		if old != fresh {
			if _, err := f.login.VerifyRecheck(ctx, c.Token, old); err == nil {
				t.Fatal("古いコードで確認できた")
			}
		}
		if _, err := f.login.VerifyRecheck(ctx, c.Token, fresh); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("コードの期限（10分）切れ", func(t *testing.T) {
		f := newLoginFixture(t)
		c := f.challenge(t, "new-device")
		code := f.lastCode(t)
		f.clock.Advance(auth.CodeTTL)
		_, err := f.login.VerifyRecheck(ctx, c.Token, code)
		wantCode(t, err, service.CodeVerificationCodeExpired)
	})

	t.Run("チャレンジの期限（15分）切れ", func(t *testing.T) {
		f := newLoginFixture(t)
		c := f.challenge(t, "new-device")
		f.clock.Advance(auth.LoginChallengeTTL - time.Minute)
		if err := f.login.ResendRecheckCode(ctx, c.Token); err != nil {
			t.Fatal(err)
		}
		code := f.lastCode(t)
		f.clock.Advance(time.Minute)
		_, err := f.login.VerifyRecheck(ctx, c.Token, code)
		wantCode(t, err, service.CodeInvalidLoginChallenge)
		wantCode(t, f.login.ResendRecheckCode(ctx, c.Token), service.CodeInvalidLoginChallenge)
	})

	t.Run("知らないチャレンジ", func(t *testing.T) {
		f := newLoginFixture(t)
		_, err := f.login.VerifyRecheck(ctx, "no-such-challenge", "123456")
		wantCode(t, err, service.CodeInvalidLoginChallenge)
		wantCode(t, f.login.ResendRecheckCode(ctx, ""), service.CodeInvalidLoginChallenge)
	})

	t.Run("確認の途中で停止されたらログインできない", func(t *testing.T) {
		f := newLoginFixture(t)
		c := f.challenge(t, "new-device")
		f.setStatus(t, domain.UserStatusSuspended)
		_, err := f.login.VerifyRecheck(ctx, c.Token, f.lastCode(t))
		wantCode(t, err, service.CodeAccountSuspended)
	})

	t.Run("同時に確認してもセッションは1つだけ", func(t *testing.T) {
		f := newLoginFixture(t)
		c := f.challenge(t, "new-device")
		code := f.lastCode(t)
		var ok atomic.Int32
		var wg sync.WaitGroup
		for range 5 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if _, err := f.login.VerifyRecheck(ctx, c.Token, code); err == nil {
					ok.Add(1)
				}
			}()
		}
		wg.Wait()
		if ok.Load() != 1 {
			t.Errorf("成功した回数=%d want=1", ok.Load())
		}
		if n := f.count(t, "sessions"); n != 2 {
			t.Errorf("sessions=%d want=2（登録時と再確認の1つずつ）", n)
		}
	})
}

func TestSessionService(t *testing.T) {
	ctx := context.Background()

	login := func(t *testing.T, f *loginFixture) string {
		t.Helper()
		res, err := f.login.Login(ctx, email, password, knownDevice)
		if err != nil || res.Session == nil {
			t.Fatalf("ログインできない: res=%+v err=%v", res, err)
		}
		return res.Session.Token
	}

	t.Run("ログアウトしたセッションは使えない", func(t *testing.T) {
		f := newLoginFixture(t)
		token := login(t, f)
		if err := f.sessions.Logout(ctx, token); err != nil {
			t.Fatal(err)
		}
		_, err := f.sessions.Authenticate(ctx, token)
		wantCode(t, err, "unauthorized")
		// 2回目・未ログインのログアウトも成功にする
		if err := f.sessions.Logout(ctx, token); err != nil {
			t.Errorf("2回目のログアウト: %v", err)
		}
		if err := f.sessions.Logout(ctx, ""); err != nil {
			t.Errorf("未ログインのログアウト: %v", err)
		}
	})

	t.Run("ほかの端末のセッションはログアウトの影響を受けない", func(t *testing.T) {
		f := newLoginFixture(t)
		a, b := login(t, f), login(t, f)
		if err := f.sessions.Logout(ctx, a); err != nil {
			t.Fatal(err)
		}
		if _, err := f.sessions.Authenticate(ctx, b); err != nil {
			t.Errorf("別のセッションまで失効した: %v", err)
		}
	})

	t.Run("30日で期限切れ", func(t *testing.T) {
		f := newLoginFixture(t)
		token := login(t, f)
		f.clock.Advance(auth.SessionTTL - time.Second)
		if _, err := f.sessions.Authenticate(ctx, token); err != nil {
			t.Fatalf("期限の直前: %v", err)
		}
		f.clock.Advance(time.Second)
		_, err := f.sessions.Authenticate(ctx, token)
		wantCode(t, err, "unauthorized")
	})

	t.Run("停止・失効・削除されたユーザーのセッションは使えない", func(t *testing.T) {
		for _, st := range []domain.UserStatus{domain.UserStatusSuspended, domain.UserStatusExpired, domain.UserStatusDeleted} {
			f := newLoginFixture(t)
			token := login(t, f)
			f.setStatus(t, st)
			_, err := f.sessions.Authenticate(ctx, token)
			wantCode(t, err, "unauthorized")
		}
	})

	t.Run("トークンがない・知らない", func(t *testing.T) {
		f := newLoginFixture(t)
		for _, token := range []string{"", "no-such-token"} {
			_, err := f.sessions.Authenticate(ctx, token)
			wantCode(t, err, "unauthorized")
		}
	})
}
