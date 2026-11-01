package server_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/auth"
	"github.com/takuma0403/osagari_web/server/internal/clock"
	"github.com/takuma0403/osagari_web/server/internal/handler"
	"github.com/takuma0403/osagari_web/server/internal/jobs"
	"github.com/takuma0403/osagari_web/server/internal/logging"
	"github.com/takuma0403/osagari_web/server/internal/server"
)

type countingJob struct{ calls int }

func (j *countingJob) Run(context.Context) error {
	j.calls++
	return nil
}

// 内部用エンドポイント（/api/internal/**）は、呼び出し元を確かめられなければ存在しないパスと同じ 404 になる（JOBS-1, ADR 0027）。
// セッションでは通らない。
func Test内部用エンドポイントは呼び出し元を確かめる(t *testing.T) {
	job := &countingJob{}
	var logs bytes.Buffer
	e := server.New(server.Deps{
		Logger:   logging.New(&logs, slog.LevelInfo),
		Clock:    clock.NewFake(time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)),
		Handler:  handler.New(handler.Deps{ShortJob: job, DailyJob: job}),
		Sessions: adminAuthenticator{},
		Jobs:     jobs.SecretVerifier{Secret: "local-secret"},
	})
	tests := []struct {
		name          string
		authorization string
		session       string
		want          int
	}{
		{"認証なし", "", "", http.StatusNotFound},
		{"シークレットが違う", "Bearer wrong", "", http.StatusNotFound},
		{"管理者のセッションだけ", "", "admin", http.StatusNotFound},
		{"シークレットが正しい", "Bearer local-secret", "", http.StatusNoContent},
	}
	for _, path := range []string{"/api/internal/jobs/short", "/api/internal/jobs/daily"} {
		for _, tt := range tests {
			t.Run(path+"/"+tt.name, func(t *testing.T) {
				before := job.calls
				req := httptest.NewRequest(http.MethodPost, path, nil)
				if tt.authorization != "" {
					req.Header.Set("Authorization", tt.authorization)
				}
				if tt.session != "" {
					req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: tt.session})
				}
				rec := httptest.NewRecorder()
				e.ServeHTTP(rec, req)
				if rec.Code != tt.want {
					t.Fatalf("status = %d, want %d: %s", rec.Code, tt.want, rec.Body)
				}
				if ran := job.calls > before; ran != (tt.want == http.StatusNoContent) {
					t.Errorf("ジョブを実行したか = %v", ran)
				}
			})
		}
	}
}
