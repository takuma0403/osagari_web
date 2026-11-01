-- ログインチャレンジ（AUTHN-3, ADR 0009）。パスワードが一致したが再確認が必要なときに発行し、コードの確認・再送に使う。
-- 15分間・一度だけ使える。トークンの値は保存せず、SHA-256 のハッシュだけを持つ（ADR 0006）
CREATE TABLE login_challenges (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL UNIQUE,
  reason TEXT NOT NULL
    CHECK (reason IN ('code_expired', 'new_device')),
  -- ログイン時に送られた端末識別子の SHA-256。コードの確認に成功したらこの端末を登録する。ヘッダーがなければ NULL
  device_hash TEXT NULL,
  expires_at TEXT NOT NULL,
  -- 使用済みにした日時。条件付きUPDATE（consumed_at IS NULL）で一度だけ使えるようにする
  consumed_at TEXT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX login_challenges_user_id_idx ON login_challenges (user_id);
