package handler_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/api"
	"github.com/takuma0403/osagari_web/server/internal/auth"
	"github.com/takuma0403/osagari_web/server/internal/clock"
	"github.com/takuma0403/osagari_web/server/internal/handler"
	"github.com/takuma0403/osagari_web/server/internal/logging"
	"github.com/takuma0403/osagari_web/server/internal/mail"
	"github.com/takuma0403/osagari_web/server/internal/repository"
	"github.com/takuma0403/osagari_web/server/internal/server"
	"github.com/takuma0403/osagari_web/server/internal/service"
	"github.com/takuma0403/osagari_web/server/internal/testutil/testdb"
)

const email = "26238001@edu.cc.saga-u.ac.jp"

func newServer(t *testing.T) (http.Handler, *mail.Recorder) {
	t.Helper()
	gdb := testdb.New(t)
	clk := clock.NewFake(time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC))
	mailer := &mail.Recorder{}
	signup := service.NewSignupService(service.SignupDeps{
		Tx:       repository.NewTransactor(gdb),
		Users:    repository.NewUserRepository(gdb),
		Codes:    repository.NewVerificationCodeRepository(gdb),
		Tokens:   repository.NewPasswordSetupTokenRepository(gdb),
		Sessions: repository.NewSessionRepository(gdb),
		Devices:  repository.NewTrustedDeviceRepository(gdb),
		Mailer:   mailer,
		Clock:    clk,
	})
	e := server.New(server.Deps{
		Logger:  logging.New(io.Discard, slog.LevelInfo),
		Clock:   clk,
		Handler: handler.New(handler.Deps{Signup: signup}),
	})
	return e, mailer
}

func post(t *testing.T, h http.Handler, path string, body any, header map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api"+path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range header {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("JSON でない: %v: %s", err, rec.Body.String())
	}
	return v
}

var codeInMail = regexp.MustCompile(`認証コード：([0-9]{6})`)

// 完了条件「登録〜ログイン状態までが API で通る」（AUTHN-2）。
func Test新規登録からログイン状態まで(t *testing.T) {
	srv, mailer := newServer(t)

	rec := post(t, srv, "/auth/signup/estimate", api.EmailInput{Email: email}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("estimate: %d %s", rec.Code, rec.Body.String())
	}
	est := decode[api.StudentEstimate](t, rec)
	if est.DepartmentName == nil || *est.DepartmentName != "理工学部 理工学科" || est.ExpiresOn.String() != "2030-03-31" {
		t.Errorf("estimate=%+v", est)
	}

	if rec := post(t, srv, "/auth/signup/codes", api.EmailInput{Email: email}, nil); rec.Code != http.StatusAccepted {
		t.Fatalf("codes: %d %s", rec.Code, rec.Body.String())
	}
	m, _ := mailer.Last()
	code := codeInMail.FindStringSubmatch(m.Body)[1]

	rec = post(t, srv, "/auth/signup/codes/verify", api.SignupCodeVerification{Email: email, Code: code}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("verify: %d %s", rec.Code, rec.Body.String())
	}
	verified := decode[api.SignupCodeVerified](t, rec)

	rec = post(t, srv, "/auth/signup/complete",
		api.SignupCompletion{SetupToken: verified.SetupToken, Password: "correct horse battery"},
		map[string]string{"X-Device-Id": "device-1"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("complete: %d %s", rec.Code, rec.Body.String())
	}
	user := decode[api.CurrentUser](t, rec)
	if user.Email != email || user.StudentNumber != "26238001" || user.Department == nil || *user.Department != "238" ||
		user.Status != api.Active || user.SellerStatus != api.None {
		t.Errorf("user=%+v", user)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("Cookie が1つではない: %v", cookies)
	}
	c := cookies[0]
	if c.Name != auth.SessionCookieName || c.Value == "" || !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode ||
		c.MaxAge != int(auth.SessionTTL/time.Second) {
		t.Errorf("Cookie=%+v", c)
	}
}

// 完了条件「登録済みアドレスでも応答が区別できない」（AUTHN-2）。
func Test登録済みのアドレスでもコード送信の応答が同じ(t *testing.T) {
	srv, mailer := newServer(t)
	register := func(email string) {
		post(t, srv, "/auth/signup/codes", api.EmailInput{Email: email}, nil)
		m, _ := mailer.Last()
		code := codeInMail.FindStringSubmatch(m.Body)[1]
		v := decode[api.SignupCodeVerified](t, post(t, srv, "/auth/signup/codes/verify", api.SignupCodeVerification{Email: email, Code: code}, nil))
		post(t, srv, "/auth/signup/complete", api.SignupCompletion{SetupToken: v.SetupToken, Password: "correct horse battery"}, nil)
	}
	register(email)

	registered := post(t, srv, "/auth/signup/codes", api.EmailInput{Email: email}, nil)
	fresh := post(t, srv, "/auth/signup/codes", api.EmailInput{Email: "26238002@edu.cc.saga-u.ac.jp"}, nil)
	if registered.Code != fresh.Code || registered.Body.String() != fresh.Body.String() ||
		len(registered.Header()) != len(fresh.Header()) {
		t.Errorf("応答が違う: 登録済み=%d %v %q / 未登録=%d %v %q",
			registered.Code, registered.Header(), registered.Body.String(), fresh.Code, fresh.Header(), fresh.Body.String())
	}

	// 誰にも届いていないコードで確認しても、未登録のときと同じ「コードが違う」になる
	m, _ := mailer.Last() // 未登録のアドレスに届いたコード。これと違う値を入力する
	wrong := "000000"
	if codeInMail.FindStringSubmatch(m.Body)[1] == wrong {
		wrong = "000001"
	}
	wrongReg := post(t, srv, "/auth/signup/codes/verify", api.SignupCodeVerification{Email: email, Code: wrong}, nil)
	wrongNew := post(t, srv, "/auth/signup/codes/verify", api.SignupCodeVerification{Email: "26238002@edu.cc.saga-u.ac.jp", Code: wrong}, nil)
	if wrongReg.Code != wrongNew.Code || decode[api.Error](t, wrongReg) != decode[api.Error](t, wrongNew) {
		t.Errorf("確認の応答が違う: %s / %s", wrongReg.Body.String(), wrongNew.Body.String())
	}
}

func Test入力の誤りはError形式の400(t *testing.T) {
	srv, _ := newServer(t)
	cases := []struct {
		name string
		path string
		body any
		code string
	}{
		{"形式が違うアドレスの推定", "/auth/signup/estimate", api.EmailInput{Email: "a@example.com"}, string(service.CodeInvalidEmail)},
		{"形式が違うアドレスへの送信", "/auth/signup/codes", api.EmailInput{Email: "a@example.com"}, string(service.CodeInvalidEmail)},
		{"短いパスワード", "/auth/signup/complete", api.SignupCompletion{SetupToken: "x", Password: "short"}, string(service.CodePasswordTooShort)},
		{"無効なトークン", "/auth/signup/complete", api.SignupCompletion{SetupToken: "x", Password: "correct horse battery"}, string(service.CodeInvalidSetupToken)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := post(t, srv, tc.path, tc.body, nil)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status=%d %s", rec.Code, rec.Body.String())
			}
			if got := decode[api.Error](t, rec); got.Code != tc.code || got.Message == "" {
				t.Errorf("error=%+v", got)
			}
		})
	}

	t.Run("JSON でない本文", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/signup/codes", bytes.NewBufferString("{"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest || decode[api.Error](t, rec).Code != "bad_request" {
			t.Errorf("status=%d %s", rec.Code, rec.Body.String())
		}
	})
}
