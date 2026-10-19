-- 商品（ITEM, DATA）。下書きはタイトルだけ必須なので、ほかの共通項目は NULL を許す（ADR 0016）。
-- 公開するときに共通項目がそろっているかは service で確かめる
CREATE TABLE items (
  id TEXT PRIMARY KEY,
  seller_id TEXT NOT NULL REFERENCES users (id),
  -- 商品ページの固定URL /items/:shareId に使う（BROWSE）
  share_id TEXT NOT NULL UNIQUE,
  status TEXT NOT NULL DEFAULT 'draft'
    CHECK (status IN ('draft', 'published', 'requested', 'in_deal', 'completed', 'withdrawn', 'suspended')),
  title TEXT NOT NULL CHECK (length(title) BETWEEN 1 AND 100),
  category TEXT NULL CHECK (category IN ('textbook', 'furniture_appliance', 'game_other')),
  condition TEXT NULL CHECK (condition IN ('new_like', 'good', 'fair')),
  -- 価格（円の整数。PAY-2）
  price INTEGER NULL CHECK (price BETWEEN 100 AND 100000),
  description TEXT NULL CHECK (length(description) <= 2000),
  campus TEXT NULL CHECK (campus IN ('honjo', 'nabeshima')),
  -- 教科書・参考書の追加項目（category が textbook のときだけ入れる）
  course_name TEXT NULL CHECK (length(course_name) <= 100),
  instructor TEXT NULL CHECK (length(instructor) <= 100),
  -- ISBN（ハイフンを除いた10桁か13桁）
  isbn TEXT NULL CHECK (length(isbn) IN (10, 13)),
  has_writing INTEGER NULL CHECK (has_writing IN (0, 1)),
  -- 家具・家電の追加項目（category が furniture_appliance のときだけ入れる）
  size TEXT NULL CHECK (length(size) <= 100),
  operation_checked INTEGER NULL CHECK (operation_checked IN (0, 1)),
  -- 最後に公開した日時（新着順。BROWSE）。一度も公開していなければ NULL
  published_at TEXT NULL,
  -- 最後に取り下げた日時
  withdrawn_at TEXT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

-- 自分の出品一覧（AUTH-3 マイページ）
CREATE INDEX items_seller_created_at_idx ON items (seller_id, created_at);

-- 公開中の商品の新着順（BROWSE）
CREATE INDEX items_status_published_at_idx ON items (status, published_at);
