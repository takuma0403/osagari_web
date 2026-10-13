package logging_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/takuma0403/osagari_web/server/internal/logging"
)

func Testマスク(t *testing.T) {
	tests := []struct {
		name    string
		log     func(l *slog.Logger)
		key     string
		want    string
		leakage string // 出力のどこにも出てはいけない文字列
	}{
		{"キー名 password", func(l *slog.Logger) { l.Info("x", "password", "hunter2hunter2") }, "password", "[REDACTED]", "hunter2"},
		{"キー名に token を含む", func(l *slog.Logger) { l.Info("x", "handover_token", "abc") }, "handover_token", "[REDACTED]", ""},
		{"キー名 code", func(l *slog.Logger) { l.Info("x", "code", "123456") }, "code", "[REDACTED]", "123456"},
		{"キー名 email", func(l *slog.Logger) { l.Info("x", "Email", "x") }, "Email", "[REDACTED]", ""},
		{"キー名 Cookie", func(l *slog.Logger) { l.Info("x", "Cookie", "__session=abc") }, "Cookie", "[REDACTED]", "__session=abc"},
		{"値の中のメール", func(l *slog.Logger) { l.Info("x", "detail", "to 24123456@edu.cc.saga-u.ac.jp failed") },
			"detail", "to [EMAIL] failed", "24123456"},
		{"値の中の学籍番号", func(l *slog.Logger) { l.Info("x", "detail", "user 24123456 not found") },
			"detail", "user [NUMBER] not found", "24123456"},
		{"エラー値", func(l *slog.Logger) {
			l.Error("x", "err", errors.New("UNIQUE constraint failed: 24123456@edu.cc.saga-u.ac.jp"))
		},
			"err", "UNIQUE constraint failed: [EMAIL]", "24123456"},
		{"グループ内のキー", func(l *slog.Logger) { l.Info("x", slog.Group("req", "password", "p@ssw0rd!!")) }, "", "", "p@ssw0rd!!"},
		{"メッセージ", func(l *slog.Logger) { l.Info("login 24123456") }, "message", "login [NUMBER]", "24123456"},
		{"request_id は伏せない", func(l *slog.Logger) { l.Info("x", "request_id", "ab12345678cd") }, "request_id", "ab12345678cd", ""},
		{"短い数字は伏せない", func(l *slog.Logger) { l.Info("x", "status", "404") }, "status", "404", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tt.log(logging.New(&buf, slog.LevelDebug))
			if tt.leakage != "" && strings.Contains(buf.String(), tt.leakage) {
				t.Fatalf("%q が出力に含まれている: %s", tt.leakage, buf.String())
			}
			if tt.key == "" {
				return
			}
			got := decode(t, &buf)[tt.key]
			if got != tt.want {
				t.Fatalf("%s=%v want=%q", tt.key, got, tt.want)
			}
		})
	}
}

func Test重大度はCloudLoggingの形式(t *testing.T) {
	tests := []struct {
		log  func(l *slog.Logger)
		want string
	}{
		{func(l *slog.Logger) { l.Debug("x") }, "DEBUG"},
		{func(l *slog.Logger) { l.Info("x") }, "INFO"},
		{func(l *slog.Logger) { l.Warn("x") }, "WARNING"},
		{func(l *slog.Logger) { l.Error("x") }, "ERROR"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			var buf bytes.Buffer
			tt.log(logging.New(&buf, slog.LevelDebug))
			m := decode(t, &buf)
			if m["severity"] != tt.want {
				t.Fatalf("severity=%v want=%s", m["severity"], tt.want)
			}
			if _, ok := m["level"]; ok {
				t.Fatal("level が残っている")
			}
		})
	}
}

func decode(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("JSON でない: %v: %s", err, buf.String())
	}
	return m
}
