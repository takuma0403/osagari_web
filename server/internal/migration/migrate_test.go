package migration_test

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

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

// 000019 が、旧ルールの初期値のままの有効・停止のユーザーだけ有効期限を在学年限に延ばし、down で戻すことを確かめる（AUTH-3, ADR 0047）。
func Test有効期限を在学年限に延ばす(t *testing.T) {
	sqlDB, err := db.Open("file:" + filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	m, err := migration.New(sqlDB, "file://"+testdb.MigrationsDir(), migration.DefaultTable)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Migrate(18); err != nil {
		t.Fatalf("18 まで: %v", err)
	}

	const ts = "2026-10-14T00:00:00.000000Z"
	users := []struct {
		id, code, expiresOn, status, want string
	}{
		{"u1", "238", "2030-03-31", "active", "2034-03-31"},    // 4年制
		{"u2", "211", "2032-03-31", "suspended", "2038-03-31"}, // 医学部医学科（6年制）
		{"u3", "726", "2030-03-31", "active", "2034-03-31"},    // 表にないコード
		{"u4", "238", "2031-03-31", "active", "2031-03-31"},    // 管理者が変えた
		{"u5", "238", "2030-03-31", "expired", "2030-03-31"},   // 失効済み
		{"u6", "238", "2030-03-31", "deleted", "2030-03-31"},   // 削除済み
	}
	for i, u := range users {
		_, err := sqlDB.Exec(`INSERT INTO users (id, email, student_number, admission_year, department_code, expires_on,
			status, password_hash, code_verified_at, created_at, updated_at)
			VALUES (?, ?, ?, 2026, ?, ?, ?, 'hash', ?, ?, ?)`,
			u.id, u.id+"@example.com", "2600000"+string(rune('0'+i)), u.code, u.expiresOn, u.status, ts, ts, ts)
		if err != nil {
			t.Fatal(err)
		}
	}

	assertExpiresOn := func(t *testing.T, want func(i int) string) {
		t.Helper()
		for i, u := range users {
			// go-libsql は日付の形の TEXT を時刻として返すので、日付にして比べる
			var at time.Time
			if err := sqlDB.QueryRow("SELECT expires_on FROM users WHERE id = ?", u.id).Scan(&at); err != nil {
				t.Fatal(err)
			}
			if got := at.Format("2006-01-02"); got != want(i) {
				t.Errorf("%s: expires_on=%s want=%s", u.id, got, want(i))
			}
		}
	}

	if err := m.Migrate(19); err != nil {
		t.Fatalf("up: %v", err)
	}
	assertExpiresOn(t, func(i int) string { return users[i].want })

	if err := m.Migrate(18); err != nil {
		t.Fatalf("down: %v", err)
	}
	assertExpiresOn(t, func(i int) string { return users[i].expiresOn })
}
