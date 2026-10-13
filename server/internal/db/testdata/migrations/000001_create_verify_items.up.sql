-- libSQL 検証用（本番スキーマではない）。1ファイルに複数文を書いても適用できるかも確認する。
CREATE TABLE verify_items (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL,
  status TEXT NOT NULL,
  price INTEGER NOT NULL,
  created_at DATETIME NOT NULL
);
CREATE UNIQUE INDEX verify_items_code_idx ON verify_items (code);
