// API サーバーの起動。設定の読み込みと依存の組み立て（DI）だけを行う。
package main

import (
	"log/slog"
	"os"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"github.com/takuma0403/osagari_web/server/internal/api"
	"github.com/takuma0403/osagari_web/server/internal/db"
	"github.com/takuma0403/osagari_web/server/internal/handler"
)

func main() {
	if err := run(); err != nil {
		slog.Error("起動に失敗しました", "err", err)
		os.Exit(1)
	}
}

func run() error {
	sqlDB, err := db.Open(mustEnv("DATABASE_URL"))
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	e := echo.New()
	e.Use(middleware.Recover())
	e.Use(middleware.RequestLogger())

	api.RegisterHandlersWithBaseURL(e, api.NewStrictHandler(handler.New(), nil), "/api")

	return e.Start(":" + envOr("PORT", "8080"))
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("環境変数が設定されていません", "key", key)
		os.Exit(1)
	}
	return v
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
