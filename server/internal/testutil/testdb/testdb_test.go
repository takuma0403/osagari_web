package testdb_test

import (
	"os"
	"testing"

	"github.com/takuma0403/osagari_web/server/internal/migration"
	"github.com/takuma0403/osagari_web/server/internal/testutil/testdb"
)

func TestMigrationsDirが存在する(t *testing.T) {
	if fi, err := os.Stat(testdb.MigrationsDir()); err != nil || !fi.IsDir() {
		t.Fatalf("migrations ディレクトリがない: %v", err)
	}
}

func TestマイグレーションDBが使える(t *testing.T) {
	gdb := testdb.New(t)

	var n int64
	if err := gdb.Raw("SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = ?", migration.DefaultTable).
		Scan(&n).Error; err != nil || n != 1 {
		t.Fatalf("%s がない: n=%d err=%v", migration.DefaultTable, n, err)
	}

	// 条件付きUPDATE の更新件数が取れること（DB）
	if err := gdb.Exec("CREATE TABLE t (id INTEGER PRIMARY KEY, status TEXT NOT NULL)").Error; err != nil {
		t.Fatal(err)
	}
	if err := gdb.Exec("INSERT INTO t (id, status) VALUES (1, 'requested')").Error; err != nil {
		t.Fatal(err)
	}
	for i, want := range []int64{1, 0} {
		res := gdb.Exec("UPDATE t SET status = 'in_deal' WHERE id = 1 AND status = 'requested'")
		if res.Error != nil || res.RowsAffected != want {
			t.Fatalf("%d回目: RowsAffected=%d want=%d err=%v", i+1, res.RowsAffected, want, res.Error)
		}
	}
}

func Testテストごとに別のDBになる(t *testing.T) {
	a, b := testdb.New(t), testdb.New(t)
	if err := a.Exec("CREATE TABLE only_a (id INTEGER)").Error; err != nil {
		t.Fatal(err)
	}
	var n int64
	b.Raw("SELECT count(*) FROM sqlite_master WHERE name = 'only_a'").Scan(&n)
	if n != 0 {
		t.Fatal("DB が共有されている")
	}
}
