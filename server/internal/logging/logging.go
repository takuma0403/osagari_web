// Package logging はアプリのロガーを作る。出力は Cloud Logging が読める JSON で、
// 学籍番号・メールアドレス・トークン・パスワード・コードがログに出ないように伏せ字にする（server/CLAUDE.md の規約）。
package logging

import (
	"io"
	"log/slog"
	"regexp"
	"strings"
)

const redacted = "[REDACTED]"

// sensitiveKeys は値をまるごと伏せるキー名（小文字で完全一致）。
var sensitiveKeys = map[string]bool{
	"code": true, "otp": true, "email": true, "student_number": true, "studentnumber": true,
}

// plainKeys は値の伏せ字をかけないキー名。乱数で作る ID（応答ヘッダーとの突き合わせに使う）が偶然伏せられないようにする。
var plainKeys = map[string]bool{"request_id": true}

// sensitiveKeyParts は、キー名に含まれていたら値をまるごと伏せる部分文字列（小文字）。
var sensitiveKeyParts = []string{"password", "token", "secret", "authorization", "cookie", "session", "verification_code"}

var (
	emailPattern = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)
	// 学籍番号（8桁）を確実に伏せるため、8桁以上の数字の並びをすべて伏せる
	digitsPattern = regexp.MustCompile(`[0-9]{8,}`)
)

// New は w に JSON でログを出すロガーを返す。
func New(w io.Writer, level slog.Leveler) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: replaceAttr,
	}))
}

func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if len(groups) == 0 {
		switch a.Key {
		case slog.TimeKey:
			return a
		case slog.LevelKey:
			// Cloud Logging は severity を重大度として読む
			return slog.String("severity", severity(a.Value))
		case slog.MessageKey:
			return slog.String("message", maskString(a.Value.String()))
		}
	}
	if isSensitiveKey(a.Key) {
		return slog.String(a.Key, redacted)
	}
	if plainKeys[a.Key] {
		return a
	}
	switch a.Value.Kind() {
	case slog.KindString:
		return slog.String(a.Key, maskString(a.Value.String()))
	case slog.KindAny:
		if err, ok := a.Value.Any().(error); ok {
			return slog.String(a.Key, maskString(err.Error()))
		}
	}
	return a
}

func isSensitiveKey(key string) bool {
	k := strings.ToLower(key)
	if sensitiveKeys[k] {
		return true
	}
	for _, p := range sensitiveKeyParts {
		if strings.Contains(k, p) {
			return true
		}
	}
	return false
}

// maskString は文字列中のメールアドレスと8桁以上の数字を伏せ字にする。
func maskString(s string) string {
	s = emailPattern.ReplaceAllString(s, "[EMAIL]")
	return digitsPattern.ReplaceAllString(s, "[NUMBER]")
}

func severity(v slog.Value) string {
	l, ok := v.Any().(slog.Level)
	if !ok {
		return v.String()
	}
	switch {
	case l >= slog.LevelError:
		return "ERROR"
	case l >= slog.LevelWarn:
		return "WARNING"
	case l >= slog.LevelInfo:
		return "INFO"
	default:
		return "DEBUG"
	}
}
