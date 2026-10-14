package auth

import (
	"context"

	"github.com/takuma0403/osagari_web/server/internal/domain"
)

// リクエストの context に載せる認証の情報。internal/server のミドルウェアが入れ、handler が読む（AUTHN-5）。

type ctxKey int

const (
	sessionTokenKey ctxKey = iota
	userKey
)

// WithSessionToken は `__session` Cookie の値を context に入れる。
func WithSessionToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, sessionTokenKey, token)
}

// SessionTokenFrom は `__session` Cookie の値を返す。Cookie がなければ空。
func SessionTokenFrom(ctx context.Context) string {
	token, _ := ctx.Value(sessionTokenKey).(string)
	return token
}

// WithUser はセッションで確かめたログイン中のユーザーを context に入れる。
func WithUser(ctx context.Context, u *domain.User) context.Context {
	return context.WithValue(ctx, userKey, u)
}

// UserFrom はログイン中のユーザーを返す。セッションが必要な操作でだけ入っている。
func UserFrom(ctx context.Context) (*domain.User, bool) {
	u, ok := ctx.Value(userKey).(*domain.User)
	return u, ok && u != nil
}
