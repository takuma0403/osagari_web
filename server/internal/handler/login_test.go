package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/api"
	"github.com/takuma0403/osagari_web/server/internal/auth"
	"github.com/takuma0403/osagari_web/server/internal/mail"
)

const password = "correct horse battery"

// signupAndLogout は deviceID の端末から登録し、ログイン状態を捨てる。
func signupAndLogout(t *testing.T, srv http.Handler, mailer *mail.Recorder, deviceID string) {
	t.Helper()
	if rec := post(t, srv, "/auth/signup/codes", api.EmailInput{Email: email}, nil); rec.Code != http.StatusAccepted {
		t.Fatalf("codes: %d %s", rec.Code, rec.Body.String())
	}
	rec := post(t, srv, "/auth/signup/codes/verify", api.SignupCodeVerification{Email: email, Code: lastCode(t, mailer)}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("verify: %d %s", rec.Code, rec.Body.String())
	}
	verified := decode[api.SignupCodeVerified](t, rec)
	rec = post(t, srv, "/auth/signup/complete", api.SignupCompletion{SetupToken: verified.SetupToken, Password: password},
		map[string]string{"X-Device-Id": deviceID})
	if rec.Code != http.StatusCreated {
		t.Fatalf("complete: %d %s", rec.Code, rec.Body.String())
	}
}

func lastCode(t *testing.T, mailer *mail.Recorder) string {
	t.Helper()
	m, ok := mailer.Last()
	if !ok {
		t.Fatal("メールが送られていない")
	}
	g := codeInMail.FindStringSubmatch(m.Body)
	if g == nil {
		t.Fatalf("メールにコードがない: %s", m.Body)
	}
	return g[1]
}

// sessionCookieOf は応答の `__session` Cookie を返す。属性（HttpOnly, Secure, SameSite=Lax）も確かめる（AUTHN-5）。
func sessionCookieOf(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name != auth.SessionCookieName {
			continue
		}
		if !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode {
			t.Errorf("Cookie の属性: %+v", c)
		}
		return c
	}
	t.Fatalf("__session がない: %v", rec.Result().Cookies())
	return nil
}

func getMe(t *testing.T, srv http.Handler, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec
}

// 完了条件「既知端末はパスワードのみ、未知端末はコード確認を経てログインできる」（AUTHN-3）。
func Testログインと再確認とログアウト(t *testing.T) {
	srv, mailer := newServer(t)
	signupAndLogout(t, srv, mailer, "device-1")

	// 既知の端末：パスワードだけでログインできる
	rec := post(t, srv, "/auth/login", api.LoginInput{Email: email, Password: password}, map[string]string{"X-Device-Id": "device-1"})
	if rec.Code != http.StatusOK {
		t.Fatalf("既知の端末: %d %s", rec.Code, rec.Body.String())
	}
	if u := decode[api.CurrentUser](t, rec); u.Email != email {
		t.Errorf("user=%+v", u)
	}
	known := sessionCookieOf(t, rec)
	if known.MaxAge != int(auth.SessionTTL/time.Second) {
		t.Errorf("Cookie の有効期間=%d", known.MaxAge)
	}

	// 未知の端末：202 とチャレンジ。この時点では Cookie を出さない
	rec = post(t, srv, "/auth/login", api.LoginInput{Email: email, Password: password}, map[string]string{"X-Device-Id": "device-2"})
	if rec.Code != http.StatusAccepted {
		t.Fatalf("未知の端末: %d %s", rec.Code, rec.Body.String())
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Errorf("再確認の前に Cookie が出た: %v", rec.Result().Cookies())
	}
	ch := decode[api.LoginChallenge](t, rec)
	if ch.Reason != api.NewDevice || ch.Email != email || ch.ChallengeToken == "" {
		t.Errorf("challenge=%+v", ch)
	}

	if rec := post(t, srv, "/auth/login/recheck/codes", api.LoginChallengeInput{ChallengeToken: ch.ChallengeToken}, nil); rec.Code != http.StatusAccepted {
		t.Fatalf("再送: %d %s", rec.Code, rec.Body.String())
	}
	rec = post(t, srv, "/auth/login/recheck/verify", api.LoginRecheckVerification{ChallengeToken: ch.ChallengeToken, Code: lastCode(t, mailer)}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("再確認: %d %s", rec.Code, rec.Body.String())
	}
	recheck := sessionCookieOf(t, rec)

	// 自分の情報
	if rec := getMe(t, srv, recheck); rec.Code != http.StatusOK || decode[api.CurrentUser](t, rec).Email != email {
		t.Fatalf("me: %d %s", rec.Code, rec.Body.String())
	}

	// ログアウトすると Cookie を消し、そのセッションは使えなくなる。ほかのセッションは使える
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(recheck)
	out := httptest.NewRecorder()
	srv.ServeHTTP(out, req)
	if out.Code != http.StatusNoContent {
		t.Fatalf("logout: %d %s", out.Code, out.Body.String())
	}
	if c := sessionCookieOf(t, out); c.Value != "" || c.MaxAge >= 0 {
		t.Errorf("Cookie が消えていない: %+v", c)
	}
	if rec := getMe(t, srv, recheck); rec.Code != http.StatusUnauthorized {
		t.Errorf("ログアウト後の me: %d", rec.Code)
	}
	if rec := getMe(t, srv, known); rec.Code != http.StatusOK {
		t.Errorf("別のセッションの me: %d", rec.Code)
	}
}

func Testログインの失敗はError形式(t *testing.T) {
	srv, mailer := newServer(t)
	signupAndLogout(t, srv, mailer, "device-1")

	tests := []struct {
		name   string
		path   string
		body   any
		status int
		code   string
	}{
		{"パスワード違い", "/auth/login", api.LoginInput{Email: email, Password: password + "x"}, http.StatusUnauthorized, "invalid_credentials"},
		{"未登録", "/auth/login", api.LoginInput{Email: "26238002@edu.cc.saga-u.ac.jp", Password: password}, http.StatusUnauthorized, "invalid_credentials"},
		{"知らないチャレンジで再送", "/auth/login/recheck/codes", api.LoginChallengeInput{ChallengeToken: "x"}, http.StatusBadRequest, "invalid_login_challenge"},
		{"知らないチャレンジで確認", "/auth/login/recheck/verify", api.LoginRecheckVerification{ChallengeToken: "x", Code: "123456"}, http.StatusBadRequest, "invalid_login_challenge"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := post(t, srv, tt.path, tt.body, nil)
			if rec.Code != tt.status {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			if e := decode[api.Error](t, rec); e.Code != tt.code || e.Message == "" {
				t.Errorf("error=%+v", e)
			}
		})
	}
}

func Test自分の情報はセッションが必要(t *testing.T) {
	srv, _ := newServer(t)
	for _, c := range []*http.Cookie{nil, {Name: auth.SessionCookieName, Value: "no-such-session"}} {
		rec := getMe(t, srv, c)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if e := decode[api.Error](t, rec); e.Code != "unauthorized" {
			t.Errorf("error=%+v", e)
		}
	}
	// 未ログインでもログアウトは 204
	if rec := post(t, srv, "/auth/logout", nil, nil); rec.Code != http.StatusNoContent {
		t.Errorf("未ログインのログアウト: %d %s", rec.Code, rec.Body.String())
	}
}
