package handler_test

import (
	"net/http"
	"testing"

	"github.com/takuma0403/osagari_web/server/internal/api"
	"github.com/takuma0403/osagari_web/server/internal/auth"
)

// 完了条件「再設定後に新パスワードでログインでき、古いセッションが無効になる」（AUTHN-4）。
func Testパスワード再設定(t *testing.T) {
	srv, mailer, clk := newServer(t)
	signupAndLogout(t, srv, mailer, "device-1")
	clk.Advance(auth.CodeSendInterval) // 登録のときのコード送信から間隔をあける（AUTHN-6）
	rec := post(t, srv, "/auth/login", api.LoginInput{Email: email, Password: password}, map[string]string{"X-Device-Id": "device-1"})
	if rec.Code != http.StatusOK {
		t.Fatalf("login: %d %s", rec.Code, rec.Body.String())
	}
	old := sessionCookieOf(t, rec)

	if rec := post(t, srv, "/auth/password-reset/codes", api.EmailInput{Email: email}, nil); rec.Code != http.StatusAccepted {
		t.Fatalf("codes: %d %s", rec.Code, rec.Body.String())
	}
	rec = post(t, srv, "/auth/password-reset/codes/verify", api.PasswordResetCodeVerification{Email: email, Code: lastCode(t, mailer)}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("verify: %d %s", rec.Code, rec.Body.String())
	}
	verified := decode[api.PasswordResetCodeVerified](t, rec)

	const newPassword = "brand new password"
	rec = post(t, srv, "/auth/password-reset/complete", api.PasswordResetCompletion{SetupToken: verified.SetupToken, Password: newPassword},
		map[string]string{"X-Device-Id": "device-2"})
	if rec.Code != http.StatusOK {
		t.Fatalf("complete: %d %s", rec.Code, rec.Body.String())
	}
	if u := decode[api.CurrentUser](t, rec); u.Email != email {
		t.Errorf("user=%+v", u)
	}
	current := sessionCookieOf(t, rec)

	if rec := getMe(t, srv, old); rec.Code != http.StatusUnauthorized {
		t.Errorf("古いセッションの me: %d", rec.Code)
	}
	if rec := getMe(t, srv, current); rec.Code != http.StatusOK {
		t.Errorf("新しいセッションの me: %d %s", rec.Code, rec.Body.String())
	}
	if rec := post(t, srv, "/auth/login", api.LoginInput{Email: email, Password: newPassword}, map[string]string{"X-Device-Id": "device-2"}); rec.Code != http.StatusOK {
		t.Errorf("新しいパスワードでのログイン: %d %s", rec.Code, rec.Body.String())
	}
	if rec := post(t, srv, "/auth/login", api.LoginInput{Email: email, Password: password}, map[string]string{"X-Device-Id": "device-1"}); rec.Code != http.StatusUnauthorized {
		t.Errorf("古いパスワードでのログイン: %d", rec.Code)
	}

	// 使用済みのトークンは Error 形式の 400
	rec = post(t, srv, "/auth/password-reset/complete", api.PasswordResetCompletion{SetupToken: verified.SetupToken, Password: newPassword}, nil)
	if rec.Code != http.StatusBadRequest || decode[api.Error](t, rec).Code != "invalid_setup_token" {
		t.Errorf("使用済みのトークン: %d %s", rec.Code, rec.Body.String())
	}
}

func Testパスワード再設定は未登録でも同じ応答(t *testing.T) {
	srv, _, _ := newServer(t)
	if rec := post(t, srv, "/auth/password-reset/codes", api.EmailInput{Email: email}, nil); rec.Code != http.StatusAccepted || rec.Body.Len() != 0 {
		t.Errorf("未登録: %d %s", rec.Code, rec.Body.String())
	}
	if rec := post(t, srv, "/auth/password-reset/codes", api.EmailInput{Email: "yamada@example.com"}, nil); rec.Code != http.StatusBadRequest {
		t.Errorf("形式違い: %d", rec.Code)
	}
}
