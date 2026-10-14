// API サーバーの起動。設定の読み込みと依存の組み立て（DI）だけを行う。
package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"gorm.io/gorm"

	"github.com/takuma0403/osagari_web/server/internal/api"
	"github.com/takuma0403/osagari_web/server/internal/clock"
	"github.com/takuma0403/osagari_web/server/internal/db"
	"github.com/takuma0403/osagari_web/server/internal/handler"
	"github.com/takuma0403/osagari_web/server/internal/logging"
	"github.com/takuma0403/osagari_web/server/internal/mail"
	"github.com/takuma0403/osagari_web/server/internal/server"
)

// config は環境変数から読む設定。一覧は .env.example。
type config struct {
	Port        string
	DatabaseURL string
	SMTP        mail.SMTPConfig
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
	mailer, err := mail.NewSMTP(cfg.SMTP, clk)
	if err != nil {
		return err
	}
	e := server.New(server.Deps{
		Logger:  logger,
		Clock:   clk,
		Handler: wire(gdb, clk, mailer),
	})
	return e.Start(":" + cfg.Port)
}

// wire は repository → service → handler の順に依存を組み立てる。機能を足すときはここに追加する。
func wire(gdb *gorm.DB, clk clock.Clock, mailer mail.Mailer) api.StrictServerInterface {
	_, _, _ = gdb, clk, mailer // 最初の repository・service を追加したら使う
	return handler.New()
}

func loadConfig() (config, error) {
	cfg := config{
		Port:        envOr("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		SMTP: mail.SMTPConfig{
			Host:     os.Getenv("SMTP_HOST"),
			Port:     envOr("SMTP_PORT", "587"),
			Username: os.Getenv("SMTP_USERNAME"),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     os.Getenv("SMTP_FROM"),
		},
	}
	if cfg.DatabaseURL == "" {
		return config{}, fmt.Errorf("環境変数 DATABASE_URL が設定されていません")
	}
	// 本番（Gmail SMTP）は STARTTLS を必須にする。無効にするのはローカルの Mailpit だけ（AUTHN-7）
	startTLS, err := strconv.ParseBool(envOr("SMTP_STARTTLS", "true"))
	if err != nil {
		return config{}, fmt.Errorf("環境変数 SMTP_STARTTLS が true / false ではありません")
	}
	cfg.SMTP.StartTLS = startTLS
	return cfg, nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
