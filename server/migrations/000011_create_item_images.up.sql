-- 商品写真（ITEM, STORAGE, DATA）。画像そのものは商品写真のバケットにあり、ここにはオブジェクト名と表示順だけを持つ。
-- 編集で写真を差し替えるときは、その商品の行を消して入れ直す
CREATE TABLE item_images (
  id TEXT PRIMARY KEY,
  item_id TEXT NOT NULL REFERENCES items (id) ON DELETE CASCADE,
  -- 商品写真のバケットのオブジェクト名（{ユーザーID}/{UUID}）。ほかの商品と同じ写真を共有させない
  object_key TEXT NOT NULL UNIQUE,
  content_type TEXT NOT NULL CHECK (content_type IN ('image/jpeg', 'image/webp')),
  size_bytes INTEGER NOT NULL CHECK (size_bytes > 0),
  -- 表示順（0 から）
  position INTEGER NOT NULL CHECK (position BETWEEN 0 AND 9),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE UNIQUE INDEX item_images_item_position_idx ON item_images (item_id, position);
