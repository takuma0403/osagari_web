-- 外部キーのある列は DROP COLUMN できないので、作り直す
CREATE TABLE settings_old (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  fee_rate_bp INTEGER NOT NULL CHECK (fee_rate_bp BETWEEN 0 AND 10000),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
INSERT INTO settings_old (id, fee_rate_bp, created_at, updated_at) SELECT id, fee_rate_bp, created_at, updated_at FROM settings;
DROP TABLE settings;
ALTER TABLE settings_old RENAME TO settings;
