-- 学生証画像の閲覧記録（CARD, STORAGE, ADMIN, ADR 0015）。管理者が閲覧用の署名付きURLを発行するたびに1行を残す。
-- 画像を削除した後も、アカウントが失効・削除された後も残すため、連鎖して消さない
CREATE TABLE student_card_views (
  id TEXT PRIMARY KEY,
  student_card_id TEXT NOT NULL REFERENCES student_cards (id),
  -- 閲覧した管理者のユーザーID
  admin_user_id TEXT NOT NULL REFERENCES users (id),
  viewed_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

-- 学生証ごとの閲覧記録を新しい順に出す（ADMIN）
CREATE INDEX student_card_views_card_viewed_at_idx ON student_card_views (student_card_id, viewed_at);
