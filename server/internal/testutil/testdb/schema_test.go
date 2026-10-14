package testdb_test

import (
	"testing"

	"github.com/takuma0403/osagari_web/server/internal/testutil/testdb"
)

const now = "2026-10-14T00:00:00Z"

func insertUser(t *testing.T, exec func(string, ...any) error, id, email, studentNumber string) error {
	t.Helper()
	return exec(`INSERT INTO users (id, email, student_number, admission_year, department_code, expires_on,
		password_hash, code_verified_at, created_at, updated_at)
		VALUES (?, ?, ?, 2026, '238', '2030-03-31', 'hash', ?, ?, ?)`,
		id, email, studentNumber, now, now, now)
}

// 認証のテーブルの制約（一意・CHECK・既定値）が効いていることを確かめる（AUTHN-2, AUTHN-5, DATA）。
func Test認証テーブルの制約(t *testing.T) {
	gdb := testdb.New(t)
	exec := func(q string, args ...any) error { return gdb.Exec(q, args...).Error }

	if err := insertUser(t, exec, "u1", "26238001@edu.cc.saga-u.ac.jp", "26238001"); err != nil {
		t.Fatalf("ユーザーを登録できない: %v", err)
	}

	var status, sellerStatus string
	if err := gdb.Raw("SELECT status, seller_status FROM users WHERE id = 'u1'").Row().Scan(&status, &sellerStatus); err != nil {
		t.Fatal(err)
	}
	if status != "active" || sellerStatus != "none" {
		t.Errorf("既定値: status=%q seller_status=%q", status, sellerStatus)
	}

	cases := []struct {
		name string
		run  func() error
	}{
		{"メールアドレスの重複", func() error {
			return insertUser(t, exec, "u2", "26238001@edu.cc.saga-u.ac.jp", "26238002")
		}},
		{"学籍番号の重複", func() error {
			return insertUser(t, exec, "u3", "26238003@edu.cc.saga-u.ac.jp", "26238001")
		}},
		{"ユーザーの状態が用語集にない値", func() error {
			return exec("UPDATE users SET status = 'unknown' WHERE id = 'u1'")
		}},
		{"出品者の承認状態が用語集にない値", func() error {
			return exec("UPDATE users SET seller_status = 'unknown' WHERE id = 'u1'")
		}},
		{"認証コードの用途が用語集にない値", func() error {
			return exec(`INSERT INTO verification_codes (id, email, purpose, code_hash, expires_at, created_at, updated_at)
				VALUES ('c1', 'a@example.com', 'unknown', 'hash', ?, ?, ?)`, now, now, now)
		}},
		{"セッションのトークンハッシュの重複", func() error {
			q := `INSERT INTO sessions (id, user_id, token_hash, expires_at, created_at, updated_at)
				VALUES (?, 'u1', 'same', ?, ?, ?)`
			if err := exec(q, "s1", now, now, now); err != nil {
				t.Fatalf("1件目のセッション: %v", err)
			}
			return exec(q, "s2", now, now, now)
		}},
		{"存在しないユーザーのセッション（外部キー）", func() error {
			return exec(`INSERT INTO sessions (id, user_id, token_hash, expires_at, created_at, updated_at)
				VALUES ('s9', 'no-such-user', 'other', ?, ?, ?)`, now, now, now)
		}},
		{"存在しないユーザーの端末（外部キー）", func() error {
			return exec(`INSERT INTO trusted_devices (id, user_id, device_hash, last_used_at, created_at, updated_at)
				VALUES ('d9', 'no-such-user', 'other', ?, ?, ?)`, now, now, now)
		}},
		{"同じユーザーの同じ端末の重複", func() error {
			q := `INSERT INTO trusted_devices (id, user_id, device_hash, last_used_at, created_at, updated_at)
				VALUES (?, 'u1', 'same', ?, ?, ?)`
			if err := exec(q, "d1", now, now, now); err != nil {
				t.Fatalf("1件目の端末: %v", err)
			}
			return exec(q, "d2", now, now, now)
		}},
		{"一時トークンの用途が想定外の値", func() error {
			return exec(`INSERT INTO password_setup_tokens (id, email, purpose, token_hash, expires_at, created_at, updated_at)
				VALUES ('p9', 'a@example.com', 'login_recheck', 'other', ?, ?, ?)`, now, now, now)
		}},
		{"一時トークンのハッシュの重複", func() error {
			q := `INSERT INTO password_setup_tokens (id, email, purpose, token_hash, expires_at, created_at, updated_at)
				VALUES (?, 'a@example.com', 'signup', 'same', ?, ?, ?)`
			if err := exec(q, "p1", now, now, now); err != nil {
				t.Fatalf("1件目の一時トークン: %v", err)
			}
			return exec(q, "p2", now, now, now)
		}},
		{"ログインチャレンジの理由が用語集にない値", func() error {
			return exec(`INSERT INTO login_challenges (id, user_id, token_hash, reason, expires_at, created_at, updated_at)
				VALUES ('l9', 'u1', 'other', 'unknown', ?, ?, ?)`, now, now, now)
		}},
		{"存在しないユーザーのログインチャレンジ（外部キー）", func() error {
			return exec(`INSERT INTO login_challenges (id, user_id, token_hash, reason, expires_at, created_at, updated_at)
				VALUES ('l8', 'no-such-user', 'other2', 'new_device', ?, ?, ?)`, now, now, now)
		}},
		{"ログインチャレンジのハッシュの重複", func() error {
			q := `INSERT INTO login_challenges (id, user_id, token_hash, reason, expires_at, created_at, updated_at)
				VALUES (?, 'u1', 'same', 'new_device', ?, ?, ?)`
			if err := exec(q, "l1", now, now, now); err != nil {
				t.Fatalf("1件目のログインチャレンジ: %v", err)
			}
			return exec(q, "l2", now, now, now)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); err == nil {
				t.Error("制約違反のはずがエラーにならない")
			}
		})
	}
}
