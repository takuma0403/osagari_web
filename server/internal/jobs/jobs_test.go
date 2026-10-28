package jobs_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/takuma0403/osagari_web/server/internal/jobs"
	"github.com/takuma0403/osagari_web/server/internal/logging"
)

// 1つの処理が失敗してもほかの処理は続け、失敗をまとめて返す（JOBS-1）。
func TestRunnerRun(t *testing.T) {
	var logs bytes.Buffer
	var ran []string
	step := func(name string, n int, err error) jobs.Step {
		return jobs.Step{Name: name, Run: func(context.Context) (int, error) {
			ran = append(ran, name)
			return n, err
		}}
	}
	boom := errors.New("DB の障害")
	err := jobs.NewShort(logging.New(&logs, slog.LevelInfo),
		step("a", 2, nil), step("b", 0, boom), step("c", 1, nil),
	).Run(context.Background())

	if !errors.Is(err, boom) || !strings.Contains(err.Error(), "b") {
		t.Errorf("err = %v, want 処理 b の失敗", err)
	}
	if strings.Join(ran, ",") != "a,b,c" {
		t.Errorf("実行した処理 = %v, want a,b,c", ran)
	}
	if got := strings.Count(logs.String(), "短周期ジョブ"); got != 2 {
		t.Errorf("件数のログ = %d 行, want 2（成功した処理だけ）\n%s", got, logs.String())
	}
}

// 日次ジョブは見出しを「日次ジョブ」にしてログに出す。
func TestDailyLogName(t *testing.T) {
	var logs bytes.Buffer
	err := jobs.NewDaily(logging.New(&logs, slog.LevelInfo),
		jobs.Step{Name: "a", Run: func(context.Context) (int, error) { return 1, nil }},
	).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(logs.String(), "日次ジョブ") {
		t.Errorf("ログに日次ジョブの見出しがない\n%s", logs.String())
	}
}

func TestSecretVerifier(t *testing.T) {
	ctx := context.Background()
	v := jobs.SecretVerifier{Secret: "local-secret"}
	tests := []struct {
		header string
		ok     bool
	}{
		{"Bearer local-secret", true},
		{"bearer local-secret", true},
		{"Bearer local-secre", false},
		{"Bearer local-secretx", false},
		{"local-secret", false},
		{"Bearer ", false},
		{"", false},
	}
	for _, tt := range tests {
		if err := v.Verify(ctx, tt.header); (err == nil) != tt.ok {
			t.Errorf("Verify(%q) = %v, want ok=%v", tt.header, err, tt.ok)
		}
	}
	// シークレットが空なら何も通さない
	if err := (jobs.SecretVerifier{}).Verify(ctx, "Bearer "); err == nil {
		t.Error("空のシークレット: err = nil")
	}
}

// OIDC の ID トークンでないものは通さない（署名の検証は idtoken に任せる）。
func TestOIDCVerifier_トークンでなければ通さない(t *testing.T) {
	v := jobs.OIDCVerifier{Audience: "https://example.run.app/api/internal/jobs/short", ServiceAccount: "scheduler@example.iam.gserviceaccount.com"}
	for _, h := range []string{"", "Bearer ", "Bearer not-a-jwt", "Basic abc"} {
		if err := v.Verify(context.Background(), h); !errors.Is(err, jobs.ErrUnauthorized) {
			t.Errorf("Verify(%q) = %v, want ErrUnauthorized", h, err)
		}
	}
}
