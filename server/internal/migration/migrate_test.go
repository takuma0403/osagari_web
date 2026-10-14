package migration_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/takuma0403/osagari_web/server/internal/db"
	"github.com/takuma0403/osagari_web/server/internal/migration"
	"github.com/takuma0403/osagari_web/server/internal/testutil/testdb"
)

// migrations/ のすべてのファイルが up → down → up で通ることを確かめる（DB）。
func Testマイグレーションをupとdownで往復できる(t *testing.T) {
	sqlDB, err := db.Open("file:" + filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	m, err := migration.New(sqlDB, "file://"+testdb.MigrationsDir(), migration.DefaultTable)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Up(); err != nil {
		t.Fatalf("up: %v", err)
	}
	wantTables := []string{"users", "sessions", "verification_codes", "trusted_devices"}
	assertTables(t, sqlDB, wantTables, true)

	if err := m.Down(); err != nil {
		t.Fatalf("down: %v", err)
	}
	assertTables(t, sqlDB, wantTables, false)

	if err := m.Up(); err != nil {
		t.Fatalf("up（2回目）: %v", err)
	}
	assertTables(t, sqlDB, wantTables, true)
}

func assertTables(t *testing.T, sqlDB *sql.DB, names []string, exist bool) {
	t.Helper()
	for _, name := range names {
		var n int
		if err := sqlDB.QueryRow("SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = ?", name).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if (n == 1) != exist {
			t.Errorf("%s: 存在する=%v、期待=%v", name, n == 1, exist)
		}
	}
}
