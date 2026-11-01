-- パスワード設定用の一時トークン（AUTHN-2, AUTHN-4）。認証コードの確認後に発行し、15分間・一度だけ使える。
-- トークンの値は保存せず、SHA-256 のハッシュだけを持つ（ADR 0006, ADR 0007）
CREATE TABLE password_setup_tokens (
  id TEXT PRIMARY KEY,
  email TEXT NOT NULL,
  purpose TEXT NOT NULL
    CHECK (purpose IN ('signup', 'password_reset')),
  token_hash TEXT NOT NULL UNIQUE,
  expires_at TEXT NOT NULL,
  -- 使用済みにした日時。条件付きUPDATE（consumed_at IS NULL）で一度だけ使えるようにする
  consumed_at TEXT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX password_setup_tokens_email_purpose_idx
  ON password_setup_tokens (email, purpose);
