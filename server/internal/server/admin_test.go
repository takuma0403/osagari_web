package server_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/clock"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/handler"
	"github.com/takuma0403/osagari_web/server/internal/logging"
	"github.com/takuma0403/osagari_web/server/internal/server"
	"github.com/takuma0403/osagari_web/server/internal/service"
)

const adminNumber = "26238099"

// adminAuthenticator は "admin" を管理者、"user" を一般ユーザー、"broken" を DB の障害として扱う。
type adminAuthenticator struct{}

func (adminAuthenticator) Authenticate(_ context.Context, token string) (*domain.User, error) {
	switch token {
	case "admin":
		return &domain.User{ID: "a1", StudentNumber: adminNumber, Status: domain.UserStatusActive}, nil
	case "user":
		return &domain.User{ID: "u1", StudentNumber: "26238001", Status: domain.UserStatusActive}, nil
	case "suspended":
		return nil, apperr.New(http.StatusForbidden, "account_suspended", "停止中")
	case "broken":
		return nil, apperr.Internal(errors.New("DB の障害"))
	}
	return nil, apperr.Unauthorized("ログインしてください")
}

// stubAdminService は呼ばれた回数を数え、管理者なら成功する。
type stubAdminService struct {
	handler.AdminStudentCardService
	admins domain.AdminList
	calls  int
}

func (s *stubAdminService) RequireAdmin(u *domain.User) error {
	s.calls++
	if !s.admins.Contains(u) {
		return apperr.NotFound("")
	}
	return nil
}

func (s *stubAdminService) ListPending(_ context.Context, u *domain.User) ([]service.AdminStudentCard, error) {
	return nil, s.RequireAdmin(u)
}

// 管理API（/api/admin/**）は、未ログイン・管理者でなければ存在しないパスと同じ 404 になる（ADR 0015）。
func Test管理APIは管理者でなければ404(t *testing.T) {
	admins, err := domain.ParseAdminList(adminNumber)
	if err != nil {
		t.Fatal(err)
	}
	svc := &stubAdminService{admins: admins}
	var logs bytes.Buffer
	srv := server.New(server.Deps{
		Logger:   logging.New(&logs, slog.LevelInfo),
		Clock:    clock.NewFake(time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)),
		Handler:  handler.New(handler.Deps{AdminStudentCard: svc}),
		Sessions: adminAuthenticator{},
		Admins:   admins,
	})
	request := func(method, target, token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, target, nil)
		if token != "" {
			req.AddCookie(&http.Cookie{Name: "__session", Value: token})
		}
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		return rec
	}

	// 存在しないパスの応答と同じであること
	missing := request(http.MethodGet, "/api/admin/missing", "")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("存在しないパス: %d", missing.Code)
	}

	targets := []struct{ method, target string }{
		{http.MethodGet, "/api/admin/session"},
		{http.MethodGet, "/api/admin/student-cards"},
		{http.MethodPost, "/api/admin/student-cards/c1/views"},
		{http.MethodGet, "/api/admin/student-cards/c1/views"},
		{http.MethodPost, "/api/admin/student-cards/c1/approve"},
		{http.MethodPost, "/api/admin/student-cards/c1/reject"},
		{http.MethodGet, "/api/admin/users"},
		{http.MethodGet, "/api/admin/users/u1"},
		{http.MethodPost, "/api/admin/users/u1/suspend"},
		{http.MethodPost, "/api/admin/users/u1/unsuspend"},
		{http.MethodPut, "/api/admin/users/u1/expires-on"},
		{http.MethodPost, "/api/admin/users/u1/delete"},
		{http.MethodGet, "/api/admin/items"},
		{http.MethodPost, "/api/admin/items/i1/suspend"},
		{http.MethodGet, "/api/admin/transactions"},
		{http.MethodPost, "/api/admin/transactions/t1/cancel"},
		{http.MethodGet, "/api/admin/settings/fee-rate"},
		{http.MethodPut, "/api/admin/settings/fee-rate"},
		// メソッド違いも 405 にしない
		{http.MethodDelete, "/api/admin/session"},
	}
	for _, token := range []string{"", "invalid", "user", "suspended"} {
		for _, tt := range targets {
			rec := request(tt.method, tt.target, token)
			if rec.Code != http.StatusNotFound || rec.Body.String() != missing.Body.String() {
				t.Errorf("%q %s %s: %d %s", token, tt.method, tt.target, rec.Code, rec.Body.String())
			}
		}
	}
	if svc.calls != 0 {
		t.Errorf("管理者でないのに service を呼んだ: %d 回", svc.calls)
	}

	// 管理者は通る（service でも判定する）
	if rec := request(http.MethodGet, "/api/admin/session", "admin"); rec.Code != http.StatusNoContent {
		t.Errorf("管理者の session: %d %s", rec.Code, rec.Body.String())
	}
	if rec := request(http.MethodGet, "/api/admin/student-cards", "admin"); rec.Code != http.StatusOK || rec.Body.String() != "[]\n" {
		t.Errorf("管理者の一覧: %d %s", rec.Code, rec.Body.String())
	}
	if svc.calls != 2 {
		t.Errorf("service の判定の回数 = %d, want 2", svc.calls)
	}

	// セッションの確認での障害は 500 のまま（404 で隠さない）
	if rec := request(http.MethodGet, "/api/admin/session", "broken"); rec.Code != http.StatusInternalServerError {
		t.Errorf("障害: %d %s", rec.Code, rec.Body.String())
	}

	// 一般の API は従来どおり 401・405
	if rec := request(http.MethodGet, "/api/student-cards/current", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("一般の API の未ログイン: %d", rec.Code)
	}
	if rec := request(http.MethodDelete, "/api/student-cards/current", "user"); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("一般の API のメソッド違い: %d", rec.Code)
	}
}
