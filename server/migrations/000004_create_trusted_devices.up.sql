-- 再確認を省略できる登録端末（AUTHN-3）。端末識別子は localStorage → ヘッダーで届く値の SHA-256 だけを持つ
CREATE TABLE trusted_devices (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  device_hash TEXT NOT NULL,
  last_used_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE (user_id, device_hash)
);
