// API サーバーの起動。設定の読み込みと依存の組み立て（DI）だけを行う。
package main

import (
	"fmt"
	"log/slog"
	"os"

	"gorm.io/gorm"

	"github.com/takuma0403/osagari_web/server/internal/api"
	"github.com/takuma0403/osagari_web/server/internal/clock"
	"github.com/takuma0403/osagari_web/server/internal/db"
	"github.com/takuma0403/osagari_web/server/internal/handler"
	"github.com/takuma0403/osagari_web/server/internal/logging"
	"github.com/takuma0403/osagari_web/server/internal/server"
)

// config は環境変数から読む設定。一覧は .env.example。
type config struct {
	Port        string
	DatabaseURL string
}

func main() {
	logger := logging.New(os.Stdout, slog.LevelInfo)
	slog.SetDefault(logger)
	if err := run(logger); err != nil {
		logger.Error("起動に失敗しました", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	sqlDB, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	gdb, err := db.NewGORM(sqlDB)
	if err != nil {
		return err
	}

	clk := clock.System()
	e := server.New(server.Deps{
		Logger:  logger,
		Clock:   clk,
		Handler: wire(gdb, clk),
	})
	return e.Start(":" + cfg.Port)
}

// wire は repository → service → handler の順に依存を組み立てる。機能を足すときはここに追加する。
func wire(gdb *gorm.DB, clk clock.Clock) api.StrictServerInterface {
	_, _ = gdb, clk // 最初の repository・service を追加したら使う
	return handler.New()
}

func loadConfig() (config, error) {
	cfg := config{
		Port:        envOr("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}
	if cfg.DatabaseURL == "" {
		return config{}, fmt.Errorf("環境変数 DATABASE_URL が設定されていません")
	}
	return cfg, nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
