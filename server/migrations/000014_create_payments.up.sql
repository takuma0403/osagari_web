-- 決済・出金の記録（PAY, JOBS-2, DATA）。フェーズ1は mock が記録するだけで、外部IDは空のまま。
-- 出金も取引ごとに1行で記録する（ADR 0021）
CREATE TABLE payments (
  id TEXT PRIMARY KEY,
  transaction_id TEXT NOT NULL REFERENCES transactions (id),
  kind TEXT NOT NULL CHECK (kind IN ('authorize', 'capture', 'release', 'refund', 'settle', 'payout')),
  -- 支払い方法。売上の確定（settle）と出金（payout）では空
  payment_method TEXT NULL CHECK (payment_method IN ('card', 'paypay')),
  -- 金額（円の整数）
  amount INTEGER NOT NULL CHECK (amount >= 0),
  status TEXT NOT NULL CHECK (status IN ('pending', 'succeeded', 'failed')),
  -- 本番の決済サービスのID（モックでは空）
  external_id TEXT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

-- 取引ごとの決済の履歴（PAY 売上管理）
CREATE INDEX payments_transaction_created_at_idx ON payments (transaction_id, created_at);
