-- メールで送る6桁の認証コード（AUTHN-1〜4, AUTHN-6）。新規登録ではユーザーがまだいないため、メールアドレスで持つ
CREATE TABLE verification_codes (
  id TEXT PRIMARY KEY,
  email TEXT NOT NULL,
  purpose TEXT NOT NULL
    CHECK (purpose IN ('signup', 'password_reset', 'login_recheck')),
  -- argon2id（PHC 形式）。平文は保存しない
  code_hash TEXT NOT NULL,
  -- 入力ミスの回数（5回を超えたら無効。AUTHN-6）
  failed_attempts INTEGER NOT NULL DEFAULT 0 CHECK (failed_attempts >= 0),
  expires_at TEXT NOT NULL,
  -- 使用済み・無効にした日時
  consumed_at TEXT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX verification_codes_email_purpose_created_at_idx
  ON verification_codes (email, purpose, created_at);
