-- ログインセッション（AUTHN-3, AUTHN-5）。Cookie `__session` の値は保存せず、SHA-256 のハッシュだけを持つ
CREATE TABLE sessions (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL UNIQUE,
  expires_at TEXT NOT NULL,
  -- ログアウト・パスワード再設定・強制停止で無効にした日時
  revoked_at TEXT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX sessions_user_id_idx ON sessions (user_id);
