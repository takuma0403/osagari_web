package server

import (
	"log/slog"

	"github.com/labstack/echo/v5"

	"github.com/takuma0403/osagari_web/server/internal/clock"
)

// accessLog は1リクエストごとに method・ルートのパターン・status・処理時間・request_id を記録する。
// 実際の URI やクエリには完了トークンや学籍番号が入りうるので出さない。
func accessLog(logger *slog.Logger, clk clock.Clock) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			start := clk.Now()
			err := next(c)
			_, status := echo.ResolveResponseStatus(c.Response(), err)
			logger.InfoContext(c.Request().Context(), "request",
				"method", c.Request().Method,
				"route", routeOf(c),
				"status", status,
				"latency_ms", clk.Now().Sub(start).Milliseconds(),
				"request_id", requestID(c),
			)
			return err
		}
	}
}

// routeOf は登録したルートのパターン（例 /api/items/:id）を返す。どのルートにも一致しなければ "unmatched"。
func routeOf(c *echo.Context) string {
	if p := c.RouteInfo().Path; p != "" {
		return p
	}
	return "unmatched"
}

// requestID は RequestID ミドルウェアが応答ヘッダーに設定した ID を返す。
func requestID(c *echo.Context) string {
	return c.Response().Header().Get(echo.HeaderXRequestID)
}
