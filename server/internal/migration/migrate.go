// Package migration は golang-migrate を libSQL の接続で動かす（DB）。スキーマ変更は migrations/ の SQL だけで行う。
package migration

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// New は libSQL の接続に golang-migrate を載せる。
// sourceURL は file://migrations のような migrations ディレクトリの URL。
func New(sqlDB *sql.DB, sourceURL string) (*migrate.Migrate, error) {
	driver, err := sqlite.WithInstance(sqlDB, &sqlite.Config{})
	if err != nil {
		return nil, fmt.Errorf("マイグレーション用ドライバを作れません: %w", err)
	}
	return migrate.NewWithDatabaseInstance(sourceURL, "libsql", driver)
}

// IsNoChange は「適用するマイグレーションがない」ことを表すエラーかを返す。
// migrations/ にファイルが1つもないとき golang-migrate は fs.ErrNotExist を返すので、これも同じ扱いにする。
func IsNoChange(err error) bool {
	return errors.Is(err, migrate.ErrNoChange) || errors.Is(err, fs.ErrNotExist)
}
