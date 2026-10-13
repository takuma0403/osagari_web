package apperr_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
)

func Testコンストラクタのステータスとコード(t *testing.T) {
	tests := []struct {
		err    *apperr.Error
		status int
		code   apperr.Code
	}{
		{apperr.BadRequest(""), http.StatusBadRequest, apperr.CodeBadRequest},
		{apperr.Unauthorized(""), http.StatusUnauthorized, apperr.CodeUnauthorized},
		{apperr.Forbidden(""), http.StatusForbidden, apperr.CodeForbidden},
		{apperr.NotFound(""), http.StatusNotFound, apperr.CodeNotFound},
		{apperr.InvalidStatusTransition(""), http.StatusConflict, apperr.CodeInvalidStatusTransition},
		{apperr.TooManyRequests(""), http.StatusTooManyRequests, apperr.CodeTooManyRequests},
		{apperr.Internal(errors.New("db down")), http.StatusInternalServerError, apperr.CodeInternal},
	}
	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			if tt.err.Status != tt.status || tt.err.StatusCode() != tt.status || tt.err.Code != tt.code {
				t.Fatalf("got status=%d code=%s want status=%d code=%s", tt.err.Status, tt.err.Code, tt.status, tt.code)
			}
			if tt.err.Message == "" {
				t.Fatal("既定のメッセージが空")
			}
		})
	}
}

func Testメッセージを指定できる(t *testing.T) {
	if got := apperr.InvalidStatusTransition("この取引はすでに完了しています").Message; got != "この取引はすでに完了しています" {
		t.Fatalf("Message=%q", got)
	}
}

func TestInternalは原因を応答メッセージに出さない(t *testing.T) {
	cause := errors.New("UNIQUE constraint failed: users.email")
	e := apperr.Internal(cause)
	if !errors.Is(e, cause) {
		t.Fatal("原因を Unwrap できない")
	}
	if e.Message == cause.Error() {
		t.Fatal("原因がメッセージに出ている")
	}
}

func TestWrapは元のエラーを変えない(t *testing.T) {
	base := apperr.NotFound("")
	wrapped := base.Wrap(errors.New("record not found"))
	if base.Err != nil {
		t.Fatal("元のエラーに原因が付いた")
	}
	if wrapped.Code != apperr.CodeNotFound || wrapped.Err == nil {
		t.Fatalf("wrapped=%+v", wrapped)
	}
}

func TestFromStatus(t *testing.T) {
	tests := []struct {
		in     int
		status int
		code   apperr.Code
	}{
		{http.StatusNotFound, http.StatusNotFound, apperr.CodeNotFound},
		{http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, apperr.CodeMethodNotAllowed},
		{http.StatusBadRequest, http.StatusBadRequest, apperr.CodeBadRequest},
		{http.StatusUnsupportedMediaType, http.StatusUnsupportedMediaType, apperr.CodeBadRequest},
		{http.StatusBadGateway, http.StatusInternalServerError, apperr.CodeInternal},
	}
	for _, tt := range tests {
		t.Run(http.StatusText(tt.in), func(t *testing.T) {
			e := apperr.FromStatus(tt.in)
			if e.Status != tt.status || e.Code != tt.code {
				t.Fatalf("got status=%d code=%s want status=%d code=%s", e.Status, e.Code, tt.status, tt.code)
			}
		})
	}
}

func TestCodeOf(t *testing.T) {
	wrapped := fmt.Errorf("approve: %w", apperr.InvalidStatusTransition(""))
	if got := apperr.CodeOf(wrapped); got != apperr.CodeInvalidStatusTransition {
		t.Fatalf("CodeOf=%q", got)
	}
	if got := apperr.CodeOf(errors.New("plain")); got != "" {
		t.Fatalf("CodeOf=%q want=空", got)
	}
}
