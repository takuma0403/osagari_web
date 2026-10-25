-- 受け渡しトークン（DEAL-3, DATA）。QR・完了リンク・完了コード。中身は取引IDと使い捨てのトークンだけで、個人情報・金額は持たない。
-- 値は保存せず、トークンは SHA-256、6桁のコードは argon2id のハッシュだけを持つ（AUTHN-2, ADR 0006）。
-- qr はトークンと確認コードの両方（5分）、link はトークンだけ、code はコードだけ（24時間）
CREATE TABLE handover_tokens (
  id TEXT PRIMARY KEY,
  transaction_id TEXT NOT NULL REFERENCES transactions (id),
  kind TEXT NOT NULL CHECK (kind IN ('qr', 'link', 'code')),
  token_hash TEXT NULL UNIQUE,
  code_hash TEXT NULL,
  expires_at TEXT NOT NULL,
  -- 使用済み・無効にした日時。条件付きUPDATE（consumed_at IS NULL）で一度だけ使えるようにする
  consumed_at TEXT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  CHECK ((token_hash IS NOT NULL) = (kind IN ('qr', 'link'))),
  CHECK ((code_hash IS NOT NULL) = (kind IN ('qr', 'code')))
);

-- 取引ごとの使えるトークン（コードの照合・再発行時の無効化）と、短周期ジョブの期限切れの削除（JOBS-1）
CREATE INDEX handover_tokens_transaction_consumed_at_idx ON handover_tokens (transaction_id, consumed_at);
CREATE INDEX handover_tokens_expires_at_idx ON handover_tokens (expires_at);
