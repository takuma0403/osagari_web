-- 設定（PAY-2, DATA）。1行だけのテーブルで、id は常に 1。
-- 手数料率はベーシスポイントの整数（500 = 5%）。取引には成立時点の値を記録する（ADR 0003）
CREATE TABLE settings (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  fee_rate_bp INTEGER NOT NULL CHECK (fee_rate_bp BETWEEN 0 AND 10000),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

-- 手数料率は未決のため、管理画面（ADMIN）で設定するまでの暫定の初期値として 5% を入れる（ADR 0017）
INSERT INTO settings (id, fee_rate_bp, created_at, updated_at)
VALUES (1, 500, '2026-10-20T00:00:00.000000Z', '2026-10-20T00:00:00.000000Z');
