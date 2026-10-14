package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/api"
	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/clock"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/handler"
	"github.com/takuma0403/osagari_web/server/internal/logging"
	"github.com/takuma0403/osagari_web/server/internal/server"
)

// stubHandler は GetHealth の振る舞いを差し替えて、エラー・panic の扱いを確かめる。
// ほかの操作は埋め込んだ nil のインターフェースのまま（呼ばない）。
type stubHandler struct {
	api.StrictServerInterface
	getHealth func() (api.GetHealthResponseObject, error)
}

func (s stubHandler) GetHealth(context.Context, api.GetHealthRequestObject) (api.GetHealthResponseObject, error) {
	return s.getHealth()
}

func newServer(t *testing.T, h api.StrictServerInterface) (http.Handler, *bytes.Buffer) {
	t.Helper()
	return newServerWithSessions(t, h, nil)
}

func newServerWithSessions(t *testing.T, h api.StrictServerInterface, sessions server.Authenticator) (http.Handler, *bytes.Buffer) {
	t.Helper()
	var logs bytes.Buffer
	e := server.New(server.Deps{
		Logger:   logging.New(&logs, slog.LevelInfo),
		Clock:    clock.NewFake(time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)),
		Handler:  h,
		Sessions: sessions,
	})
	return e, &logs
}

func do(t *testing.T, h http.Handler, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) api.Error {
	t.Helper()
	var body api.Error
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("Error 形式でない: %v: %s", err, rec.Body.String())
	}
	if body.Code == "" || body.Message == "" {
		t.Fatalf("code・message が空: %s", rec.Body.String())
	}
	return body
}

func TestHealth(t *testing.T) {
	srv, _ := newServer(t, handler.New(handler.Deps{}))
	rec := do(t, srv, http.MethodGet, "/api/health")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body api.Health
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Status != api.Ok {
		t.Fatalf("body=%s err=%v", rec.Body.String(), err)
	}
	if rec.Header().Get("X-Request-Id") == "" {
		t.Fatal("X-Request-Id が付いていない")
	}
}

func Test未実装のパスはError形式(t *testing.T) {
	srv, _ := newServer(t, handler.New(handler.Deps{}))
	tests := []struct {
		method, target string
		status         int
		code           apperr.Code
	}{
		{http.MethodGet, "/api/unknown", http.StatusNotFound, apperr.CodeNotFound},
		{http.MethodGet, "/", http.StatusNotFound, apperr.CodeNotFound},
		{http.MethodPost, "/api/health", http.StatusMethodNotAllowed, apperr.CodeMethodNotAllowed},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.target, func(t *testing.T) {
			rec := do(t, srv, tt.method, tt.target)
			if rec.Code != tt.status {
				t.Fatalf("status=%d want=%d", rec.Code, tt.status)
			}
			if got := decodeError(t, rec).Code; got != string(tt.code) {
				t.Fatalf("code=%s want=%s", got, tt.code)
			}
		})
	}
}

func Testハンドラーのエラーの変換(t *testing.T) {
	const internal = "UNIQUE constraint failed: users.email"
	tests := []struct {
		name   string
		fn     func() (api.GetHealthResponseObject, error)
		status int
		code   apperr.Code
	}{
		{"apperr はそのまま",
			func() (api.GetHealthResponseObject, error) {
				return nil, apperr.InvalidStatusTransition("この取引はすでに完了しています")
			},
			http.StatusConflict, apperr.CodeInvalidStatusTransition},
		{"包まれた apperr",
			func() (api.GetHealthResponseObject, error) {
				return nil, errors.Join(errors.New("ctx"), apperr.Forbidden(""))
			},
			http.StatusForbidden, apperr.CodeForbidden},
		{"普通のエラーは 500",
			func() (api.GetHealthResponseObject, error) { return nil, errors.New(internal) },
			http.StatusInternalServerError, apperr.CodeInternal},
		{"apperr.Internal の原因は出さない",
			func() (api.GetHealthResponseObject, error) { return nil, apperr.Internal(errors.New(internal)) },
			http.StatusInternalServerError, apperr.CodeInternal},
		{"panic は 500",
			func() (api.GetHealthResponseObject, error) { panic(internal) },
			http.StatusInternalServerError, apperr.CodeInternal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, logs := newServer(t, stubHandler{getHealth: tt.fn})
			rec := do(t, srv, http.MethodGet, "/api/health")
			if rec.Code != tt.status {
				t.Fatalf("status=%d want=%d body=%s", rec.Code, tt.status, rec.Body.String())
			}
			body := decodeError(t, rec)
			if body.Code != string(tt.code) {
				t.Fatalf("code=%s want=%s", body.Code, tt.code)
			}
			if strings.Contains(rec.Body.String(), "UNIQUE") {
				t.Fatalf("内部情報が応答に出ている: %s", rec.Body.String())
			}
			if tt.status >= 500 && !strings.Contains(logs.String(), "UNIQUE") {
				t.Fatalf("5xx の原因がログに残っていない: %s", logs.String())
			}
		})
	}
}

func Testアクセスログ(t *testing.T) {
	srv, logs := newServer(t, handler.New(handler.Deps{}))
	rec := do(t, srv, http.MethodGet, "/api/deals/24123456/complete?token=secret-token-value")

	var entry map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(logs.Bytes()), &entry); err != nil {
		t.Fatalf("ログが1行の JSON でない: %v: %s", err, logs.String())
	}
	for _, leak := range []string{"secret-token-value", "24123456", "complete"} {
		if strings.Contains(logs.String(), leak) {
			t.Fatalf("実URIの一部 %q がログに出ている: %s", leak, logs.String())
		}
	}
	want := map[string]any{
		"message":    "request",
		"method":     "GET",
		"route":      "unmatched",
		"status":     float64(http.StatusNotFound),
		"request_id": rec.Header().Get("X-Request-Id"),
	}
	for k, v := range want {
		if entry[k] != v {
			t.Errorf("%s=%v want=%v", k, entry[k], v)
		}
	}

	logs.Reset()
	do(t, srv, http.MethodGet, "/api/health")
	if !strings.Contains(logs.String(), `"route":"/api/health"`) {
		t.Fatalf("ルートのパターンが出ていない: %s", logs.String())
	}
}

// stubAuthenticator は受け取ったトークンを記録し、"valid" だけを通す。
type stubAuthenticator struct {
	tokens []string
}

func (a *stubAuthenticator) Authenticate(_ context.Context, token string) (*domain.User, error) {
	a.tokens = append(a.tokens, token)
	if token != "valid" {
		return nil, apperr.Unauthorized("ログインしてください")
	}
	return &domain.User{ID: "u1", Email: "26238001@edu.cc.saga-u.ac.jp", Status: domain.UserStatusActive}, nil
}

// セッションが要らないのは OpenAPI で security: [] の操作だけ。それ以外は __session を確かめる（AUTHN-5）。
func Testセッションの検証はAPI契約に従う(t *testing.T) {
	authn := &stubAuthenticator{}
	srv, _ := newServerWithSessions(t, handler.New(handler.Deps{}), authn)

	if rec := do(t, srv, http.MethodGet, "/api/health"); rec.Code != http.StatusOK {
		t.Fatalf("health: %d", rec.Code)
	}
	if len(authn.tokens) != 0 {
		t.Errorf("security: [] の操作でセッションを確かめた: %v", authn.tokens)
	}

	rec := do(t, srv, http.MethodGet, "/api/me")
	if rec.Code != http.StatusUnauthorized || decodeError(t, rec).Code != "unauthorized" {
		t.Fatalf("Cookie なし: %d %s", rec.Code, rec.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: "__session", Value: "valid"})
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("有効なセッション: %d %s", rec.Code, rec.Body.String())
	}
	var me api.CurrentUser
	if err := json.Unmarshal(rec.Body.Bytes(), &me); err != nil || me.Id != "u1" {
		t.Errorf("me=%s err=%v", rec.Body.String(), err)
	}
	if want := []string{"", "valid"}; strings.Join(authn.tokens, ",") != strings.Join(want, ",") {
		t.Errorf("確かめたトークン=%q want=%q", authn.tokens, want)
	}
}
