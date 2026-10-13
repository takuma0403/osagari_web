package db

import (
	"database/sql"

	// GORM の方言（SQL の組み立て）だけを借り、接続は go-libsql の *sql.DB を渡す。
	// gorm.io/driver/sqlite は mattn/go-sqlite3（cgo）と go-libsql の SQLite シンボルが衝突してリンクできず、
	// glebarez/sqlite は golang-migrate が読み込む modernc.org/sqlite とドライバ名 "sqlite" が衝突するため使わない。
	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
)

func dialector(sqlDB *sql.DB) gorm.Dialector {
	return gormlite.OpenDB(sqlDB)
}
