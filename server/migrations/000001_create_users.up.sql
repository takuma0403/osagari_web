-- ユーザー（DATA, AUTH-1〜4, AUTHN-2, AUTHN-3, AUTHN-5）
CREATE TABLE users (
  id TEXT PRIMARY KEY,
  email TEXT NOT NULL UNIQUE,
  -- 学籍番号（メールアドレスの@より前、8桁）
  student_number TEXT NOT NULL UNIQUE,
  admission_year INTEGER NOT NULL,
  -- 学籍番号の3〜5桁目をそのまま記録する
  department_code TEXT NOT NULL,
  -- 学部・学科（AUTH-4 の表で確定したコード）。NULL は「不明」。管理者が後から直せる
  department TEXT NULL,
  -- 有効期限（YYYY-MM-DD、3月31日）
  expires_on TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active'
    CHECK (status IN ('active', 'suspended', 'expired', 'deleted')),
  seller_status TEXT NOT NULL DEFAULT 'none'
    CHECK (seller_status IN ('none', 'pending', 'approved', 'rejected')),
  -- argon2id（PHC 形式）
  password_hash TEXT NOT NULL,
  -- 最後にメールのコードで確認した日時（30日での再確認に使う。AUTHN-3）
  code_verified_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX users_status_idx ON users (status);
