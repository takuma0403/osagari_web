// libSQL（Turso）で GORM と golang-migrate が動くかの検証（TECH-OPEN）。
// TEST_DATABASE_URL を設定したときだけ、compose の db に対して実行する（未設定ならスキップ）：
//
//	docker compose run --rm -e TEST_DATABASE_URL=http://db:8080 api go test -v ./internal/db/...
//
// ビルドタグで分けると gopls がこのファイルを解析しないため、環境変数で切り替える。
package db_test

import (
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"gorm.io/gorm"

	"github.com/takuma0403/osagari_web/server/internal/db"
	"github.com/takuma0403/osagari_web/server/internal/migration"
)

type verifyItem struct {
	ID        int64
	Code      string
	Status    string
	Price     int64
	CreatedAt time.Time
}

func (verifyItem) TableName() string { return "verify_items" }

func TestLibSQL(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL が未設定")
	}
	sqlDB, err := db.Open(url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	// 開発用 DB の schema_migrations を汚さないよう、検証用の管理テーブルを使う
	// 前回の失敗で残っていても始められるようにする
	for _, q := range []string{"DROP TABLE IF EXISTS verify_items", "DROP TABLE IF EXISTS verify_schema_migrations"} {
		if _, err := sqlDB.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	m, err := migration.New(sqlDB, "file://testdata/migrations", "verify_schema_migrations")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Up(); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	t.Cleanup(func() {
		if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			t.Errorf("migrate down: %v", err)
		}
		var n int
		sqlDB.QueryRow("SELECT count(*) FROM sqlite_master WHERE name = 'verify_items'").Scan(&n)
		if n != 0 {
			t.Errorf("down の2文目（DROP TABLE）が適用されていない")
		}
		if _, err := sqlDB.Exec("DROP TABLE IF EXISTS verify_schema_migrations"); err != nil {
			t.Errorf("drop verify_schema_migrations: %v", err)
		}
	})

	gdb, err := db.NewGORM(sqlDB)
	if err != nil {
		t.Fatalf("gorm: %v", err)
	}

	t.Run("migrate のバージョン", func(t *testing.T) {
		v, dirty, err := m.Version()
		if err != nil || v != 1 || dirty {
			t.Fatalf("version=%d dirty=%t err=%v", v, dirty, err)
		}
	})

	t.Run("1ファイルの複数の文がすべて適用される", func(t *testing.T) {
		var n int
		err := sqlDB.QueryRow("SELECT count(*) FROM sqlite_master WHERE type = 'index' AND name = 'verify_items_code_idx'").Scan(&n)
		if err != nil || n != 1 {
			t.Fatalf("2文目の CREATE INDEX が適用されていない: n=%d err=%v", n, err)
		}
	})

	t.Run("GORM で作成と取得", func(t *testing.T) {
		now := time.Date(2026, 10, 14, 9, 30, 0, 0, time.UTC)
		in := verifyItem{Code: "create", Status: "listed", Price: 1500, CreatedAt: now}
		if err := gdb.Create(&in).Error; err != nil {
			t.Fatal(err)
		}
		if in.ID == 0 {
			t.Fatal("ID が採番されていない")
		}
		var got verifyItem
		if err := gdb.First(&got, in.ID).Error; err != nil {
			t.Fatal(err)
		}
		if got.Code != in.Code || got.Price != in.Price || !got.CreatedAt.Equal(now) {
			t.Fatalf("got=%+v want=%+v", got, in)
		}
	})

	t.Run("条件付きUPDATE の更新件数", func(t *testing.T) {
		it := verifyItem{Code: "cond", Status: "requested", Price: 100, CreatedAt: time.Now().UTC()}
		gdb.Create(&it)
		for i, want := range []int64{1, 0} {
			res := gdb.Model(&verifyItem{}).
				Where("id = ? AND status = ?", it.ID, "requested").
				Update("status", "in_deal")
			if res.Error != nil {
				t.Fatal(res.Error)
			}
			if res.RowsAffected != want {
				t.Fatalf("%d回目: RowsAffected=%d want=%d", i+1, res.RowsAffected, want)
			}
		}
	})

	t.Run("条件付きUPDATE を並行に実行しても1件だけ成功する", func(t *testing.T) {
		it := verifyItem{Code: "race", Status: "requested", Price: 100, CreatedAt: time.Now().UTC()}
		gdb.Create(&it)
		const n = 10
		var wg sync.WaitGroup
		var mu sync.Mutex
		var success int64
		for range n {
			wg.Add(1)
			go func() {
				defer wg.Done()
				res := gdb.Model(&verifyItem{}).
					Where("id = ? AND status = ?", it.ID, "requested").
					Update("status", "in_deal")
				if res.Error != nil {
					t.Error(res.Error)
					return
				}
				mu.Lock()
				success += res.RowsAffected
				mu.Unlock()
			}()
		}
		wg.Wait()
		if success != 1 {
			t.Fatalf("成功件数=%d want=1", success)
		}
	})

	t.Run("トランザクションのロールバックとコミット", func(t *testing.T) {
		errAbort := errors.New("abort")
		err := gdb.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&verifyItem{Code: "rollback", Status: "listed", CreatedAt: time.Now().UTC()}).Error; err != nil {
				return err
			}
			return errAbort
		})
		if !errors.Is(err, errAbort) {
			t.Fatalf("err=%v", err)
		}
		if n := count(t, gdb, "rollback"); n != 0 {
			t.Fatalf("ロールバックされていない: %d件", n)
		}

		err = gdb.Transaction(func(tx *gorm.DB) error {
			return tx.Create(&verifyItem{Code: "commit", Status: "listed", CreatedAt: time.Now().UTC()}).Error
		})
		if err != nil {
			t.Fatal(err)
		}
		if n := count(t, gdb, "commit"); n != 1 {
			t.Fatalf("コミットされていない: %d件", n)
		}
	})

	t.Run("一意制約違反はエラーになる", func(t *testing.T) {
		gdb.Create(&verifyItem{Code: "dup", Status: "listed", CreatedAt: time.Now().UTC()})
		err := gdb.Create(&verifyItem{Code: "dup", Status: "listed", CreatedAt: time.Now().UTC()}).Error
		if err == nil {
			t.Fatal("一意制約違反なのにエラーにならない")
		}
		// gorm.ErrDuplicatedKey への変換は方言のエラー型に依存するため、結果だけ記録する
		t.Logf("ErrDuplicatedKey に変換されるか: %t（err=%v）", errors.Is(err, gorm.ErrDuplicatedKey), err)
	})
}

func count(t *testing.T, gdb *gorm.DB, code string) int64 {
	t.Helper()
	var n int64
	if err := gdb.Model(&verifyItem{}).Where("code = ?", code).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

// sqld（Turso）の Hrana ストリームは、使われないまま約10秒たつとサーバー側で捨てられる。
// go-libsql は捨てられたストリームを持つ接続を壊れたと報告しないため、プールに残すと以後のクエリがすべて失敗する（#60）。
func TestLibSQLIdleStream(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL が未設定")
	}
	sqlDB, err := db.Open(url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	query := func() error {
		var n int
		return sqlDB.QueryRow("SELECT 1").Scan(&n)
	}
	if err := query(); err != nil {
		t.Fatal(err)
	}
	// ストリームの期限（約10秒）より長くあける
	time.Sleep(15 * time.Second)
	for i := range 3 {
		if err := query(); err != nil {
			t.Fatalf("あけた後の%d回目のクエリが失敗した: %v", i+1, err)
		}
	}
}
