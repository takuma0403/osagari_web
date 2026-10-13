// Package db は Turso（libSQL）への接続を組み立てる（DB）。
// ローカルでは compose の libsql-server（sqld）、本番では Turso に同じドライバで接続する。
package db

import (
	"database/sql"
	"fmt"

	_ "github.com/tursodatabase/go-libsql" // "libsql" ドライバを登録する
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open は libSQL の接続を開いて疎通を確認する。
// url は http://db:8080（sqld）、libsql://<db>.turso.io?authToken=…、file:… のいずれか。
func Open(url string) (*sql.DB, error) {
	sqlDB, err := sql.Open("libsql", url)
	if err != nil {
		return nil, fmt.Errorf("libsql を開けません: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("libsql に接続できません: %w", err)
	}
	return sqlDB, nil
}

// NewGORM は開いた *sql.DB の上に GORM を載せる。スキーマ変更は golang-migrate だけで行い、AutoMigrate は使わない（DB）。
func NewGORM(sqlDB *sql.DB) (*gorm.DB, error) {
	return gorm.Open(dialector(sqlDB), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
}
