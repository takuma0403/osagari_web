-- 学生証（CARD, STORAGE, DATA, ADR 0014）。画像は非公開バケットに置き、ここにはオブジェクト名だけを持つ。
-- 再提出しても前の行と画像は消さずに superseded_at を入れて残し、アカウントの失効・削除から3年で消す。
-- アカウントの失効・削除後も残すため、users の行が消えても連鎖して消さない
CREATE TABLE student_cards (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users (id),
  -- 学生証バケットのオブジェクト名（{ユーザーID}/{UUID}）
  object_key TEXT NOT NULL UNIQUE,
  content_type TEXT NOT NULL CHECK (content_type IN ('image/jpeg', 'image/webp')),
  size_bytes INTEGER NOT NULL CHECK (size_bytes > 0),
  status TEXT NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending', 'approved', 'rejected')),
  submitted_at TEXT NOT NULL,
  -- 審査した管理者と日時（ADMIN）。審査前は NULL
  reviewed_by TEXT NULL,
  reviewed_at TEXT NULL,
  -- 再提出で置き換えた日時。NULL の行が本人の現在の学生証
  superseded_at TEXT NULL,
  -- 画像の削除予定日（YYYY-MM-DD、アカウントの失効・削除日 + 3年）。失効・削除するまでは NULL
  delete_after TEXT NULL,
  -- 日次ジョブで画像を削除した日時（JOBS-1）
  image_deleted_at TEXT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

-- 現在の学生証は1人1件
CREATE UNIQUE INDEX student_cards_current_user_idx ON student_cards (user_id) WHERE superseded_at IS NULL;

-- 審査待ちの古い順の一覧（ADMIN）
CREATE INDEX student_cards_status_submitted_at_idx ON student_cards (status, submitted_at);

-- 削除予定日を過ぎた画像の削除（JOBS-1）
CREATE INDEX student_cards_delete_after_idx ON student_cards (delete_after);
