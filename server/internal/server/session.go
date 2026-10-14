package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/labstack/echo/v5"

	"github.com/takuma0403/osagari_web/server/internal/api"
	"github.com/takuma0403/osagari_web/server/internal/auth"
	"github.com/takuma0403/osagari_web/server/internal/domain"
)

// Authenticator は `__session` のトークンからログイン中のユーザーを確かめる（service.SessionService）。
type Authenticator interface {
	Authenticate(ctx context.Context, token string) (*domain.User, error)
}

// sessionMiddleware は `__session` Cookie の値を context に入れ、セッションが必要な操作ではユーザーを確かめて context に入れる（AUTHN-5）。
// セッションが要らない操作は、OpenAPI で `security: []` を付けたものだけ（API 契約が正本。それ以外はすべてセッション必須）。
func sessionMiddleware(authn Authenticator, public map[string]bool) api.StrictMiddlewareFunc {
	return func(f api.StrictHandlerFunc, operationID string) api.StrictHandlerFunc {
		required := !public[operationID]
		return func(c *echo.Context, req any) (any, error) {
			ctx := c.Request().Context()
			token := ""
			if ck, err := c.Cookie(auth.SessionCookieName); err == nil {
				token = ck.Value
			}
			ctx = auth.WithSessionToken(ctx, token)
			if required {
				u, err := authn.Authenticate(ctx, token)
				if err != nil {
					return nil, err
				}
				ctx = auth.WithUser(ctx, u)
			}
			c.SetRequest(c.Request().WithContext(ctx))
			return f(c, req)
		}
	}
}

// publicOperations は、埋め込んだ OpenAPI から `security: []`（セッション不要）の操作を集める。
// キーは strict server がミドルウェアに渡す名前（operationId の先頭を大文字にしたもの。例 getHealth → GetHealth）。
func publicOperations() (map[string]bool, error) {
	spec, err := api.GetSwagger()
	if err != nil {
		return nil, fmt.Errorf("server: OpenAPI を読めません: %w", err)
	}
	public := map[string]bool{}
	for _, item := range spec.Paths.Map() {
		for _, op := range item.Operations() {
			if op.OperationID == "" || op.Security == nil || len(*op.Security) > 0 {
				continue
			}
			public[strings.ToUpper(op.OperationID[:1])+op.OperationID[1:]] = true
		}
	}
	return public, nil
}
