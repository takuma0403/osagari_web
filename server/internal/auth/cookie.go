package auth

import (
	"net/http"
	"time"
)

// SessionCookieName はセッションの Cookie 名。Firebase Hosting は `__session` 以外の Cookie を Cloud Run に渡さない（API, AUTHN-5）。
const SessionCookieName = "__session"

// NewSessionCookie はセッションのトークンを入れる Cookie を作る（HttpOnly, Secure, SameSite=Lax。AUTHN-5）。
// 有効期限はセッションの expiresAt に揃える。
func NewSessionCookie(token string, now, expiresAt time.Time) *http.Cookie {
	maxAge := int(expiresAt.Sub(now) / time.Second)
	if maxAge <= 0 {
		return ClearSessionCookie()
	}
	return sessionCookie(token, maxAge, expiresAt)
}

// ClearSessionCookie はブラウザからセッションの Cookie を消す Cookie を作る（ログアウトなど）。
func ClearSessionCookie() *http.Cookie {
	return sessionCookie("", -1, time.Unix(0, 0).UTC())
}

func sessionCookie(value string, maxAge int, expires time.Time) *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		Expires:  expires.UTC(),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
}
