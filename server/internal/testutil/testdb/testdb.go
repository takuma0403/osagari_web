// Package testdb は repository のテスト用に、一時ファイルの SQLite へ migrations/ を当てた DB を用意する（DB）。
// 接続は本番と同じ go-libsql・gormlite・internal/migration を通す。
package testdb

import (
	"database/sql"
	"path/filepath"
	"runtime"
	"testing"

	"gorm.io/gorm"

	"github.com/takuma0403/osagari_web/server/internal/db"
	"github.com/takuma0403/osagari_web/server/internal/migration"
)

// New はテストごとに新しい DB を作り、migrations/ をすべて適用した GORM を返す。テスト終了時に閉じる。
func New(t testing.TB) *gorm.DB {
	t.Helper()
	gdb, err := db.NewGORM(Open(t))
	if err != nil {
		t.Fatalf("testdb: gorm: %v", err)
	}
	return gdb
}

// Open は New と同じ DB を *sql.DB のまま返す。GORM を通さずに確かめたいときに使う。
func Open(t testing.TB) *sql.DB {
	t.Helper()
	sqlDB, err := db.Open("file:" + filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("testdb: open: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	// ファイルの SQLite は同時に書き込むと "database is locked" になる。本番の sqld（Turso）は書き込みを直列化するので、
	// テストでも接続を1本にして直列化する。goroutine で並行に呼ぶテスト（条件付きUPDATE の二重処理）はこの上で文単位に交互に実行される。
	sqlDB.SetMaxOpenConns(1)

	m, err := migration.New(sqlDB, "file://"+MigrationsDir(), migration.DefaultTable)
	if err != nil {
		t.Fatalf("testdb: migration: %v", err)
	}
	if err := m.Up(); err != nil && !migration.IsNoChange(err) {
		t.Fatalf("testdb: migrate up: %v", err)
	}
	return sqlDB
}

// MigrationsDir は server/migrations の絶対パスを返す。どのパッケージのテストから呼んでも同じ場所を指す。
func MigrationsDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "migrations")
}
