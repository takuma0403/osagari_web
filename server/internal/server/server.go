// Package server は Echo の組み立て（ミドルウェア・エラー応答・ルート登録）をまとめる。
// 依存の生成は cmd/api/main.go が行い、ここには出来上がったものを渡す。
package server

import (
	"log/slog"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"github.com/takuma0403/osagari_web/server/internal/api"
	"github.com/takuma0403/osagari_web/server/internal/clock"
)

// BasePath は API のパス接頭辞。Firebase Hosting の rewrites で /api/** を転送する（API）。
const BasePath = "/api"

// Deps は HTTP サーバーが使う依存。
type Deps struct {
	Logger  *slog.Logger
	Clock   clock.Clock
	Handler api.StrictServerInterface
}

// New はミドルウェアとエラーハンドラーを設定し、生成したルートを登録した Echo を返す。
func New(d Deps) *echo.Echo {
	e := echo.New()
	e.Logger = d.Logger
	e.HTTPErrorHandler = errorHandler(d.Logger)

	// 外側から順に：リクエストID → アクセスログ → panic 回復（panic も 500 としてログに残す）
	e.Use(middleware.RequestID())
	e.Use(accessLog(d.Logger, d.Clock))
	e.Use(middleware.Recover())

	api.RegisterHandlersWithBaseURL(e, api.NewStrictHandler(d.Handler, nil), BasePath)
	return e
}
