// Package migration は golang-migrate を libSQL の接続で動かす（DB）。スキーマ変更は migrations/ の SQL だけで行う。
package migration

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/libsql/sqlite-antlr4-parser/sqliteparserutils"
)

// DefaultTable はマイグレーションの適用状況を記録するテーブル名。
const DefaultTable = "schema_migrations"

// New は libSQL の接続に golang-migrate を載せる。
// sourceURL は file://migrations のような migrations ディレクトリの URL、table は適用状況を記録するテーブル名。
func New(sqlDB *sql.DB, sourceURL, table string) (*migrate.Migrate, error) {
	driver, err := sqlite.WithInstance(sqlDB, &sqlite.Config{MigrationsTable: table})
	if err != nil {
		return nil, fmt.Errorf("マイグレーション用ドライバを作れません: %w", err)
	}
	return migrate.NewWithDatabaseInstance(sourceURL, "libsql", &libsqlDriver{Driver: driver, db: sqlDB})
}

// IsNoChange は「適用するマイグレーションがない」ことを表すエラーかを返す。
// migrations/ にファイルが1つもないとき golang-migrate は fs.ErrNotExist を返すので、これも同じ扱いにする。
func IsNoChange(err error) bool {
	return errors.Is(err, migrate.ErrNoChange) || errors.Is(err, fs.ErrNotExist)
}

// libsqlDriver は golang-migrate の sqlite ドライバのうち、マイグレーションの実行（Run）だけを差し替える。
// go-libsql のリモート接続は1回の Exec に複数の文を渡すと2文目以降を黙って捨てるため、
// go-libsql と同じパーサーで文に分けて、1つのトランザクションの中で1文ずつ実行する。
type libsqlDriver struct {
	database.Driver
	db *sql.DB
}

func (d *libsqlDriver) Run(migration io.Reader) error {
	body, err := io.ReadAll(migration)
	if err != nil {
		return err
	}
	stmts, info := sqliteparserutils.SplitStatement(string(body))
	if info.IncompleteCreateTriggerStatement || info.IncompleteMultilineComment {
		return &database.Error{Err: "SQL の文が途中で終わっています", Query: body}
	}

	tx, err := d.db.Begin()
	if err != nil {
		return &database.Error{OrigErr: err, Err: "transaction start failed"}
	}
	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt); err != nil {
			if errRollback := tx.Rollback(); errRollback != nil {
				err = errors.Join(err, errRollback)
			}
			return &database.Error{OrigErr: err, Query: []byte(stmt)}
		}
	}
	if err := tx.Commit(); err != nil {
		return &database.Error{OrigErr: err, Err: "transaction commit failed"}
	}
	return nil
}
