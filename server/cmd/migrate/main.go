// マイグレーションの実行（golang-migrate を libSQL の接続で動かす）。
//
//	go run ./cmd/migrate up        … 未適用のものをすべて適用
//	go run ./cmd/migrate down      … 1つ戻す
//	go run ./cmd/migrate version   … 現在のバージョンを表示
package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/takuma0403/osagari_web/server/internal/db"
	"github.com/takuma0403/osagari_web/server/internal/migration"
)

func main() {
	if err := run(); err != nil {
		slog.Error("マイグレーションに失敗しました", "err", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return errors.New("使い方: migrate up|down|version")
	}
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return errors.New("DATABASE_URL が設定されていません")
	}
	source := os.Getenv("MIGRATIONS_URL")
	if source == "" {
		source = "file://migrations"
	}

	sqlDB, err := db.Open(url)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	m, err := migration.New(sqlDB, source)
	if err != nil {
		return err
	}

	switch os.Args[1] {
	case "up":
		err = m.Up()
	case "down":
		err = m.Steps(-1)
	case "version":
		v, dirty, verr := m.Version()
		if verr != nil {
			return verr
		}
		fmt.Printf("version=%d dirty=%t\n", v, dirty)
		return nil
	default:
		return fmt.Errorf("不明なコマンド: %s", os.Args[1])
	}
	if migration.IsNoChange(err) {
		slog.Info("適用するマイグレーションはありません")
		return nil
	}
	return err
}
