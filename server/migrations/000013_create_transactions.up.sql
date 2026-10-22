-- 取引（DEAL, DATA）。購入希望の段階から1件の取引として作る（requested）。
-- 状態遷移は条件付きUPDATE（WHERE status = ?）で行う（DB）。受け渡しトークンは handover_tokens に持つ
CREATE TABLE transactions (
  id TEXT PRIMARY KEY,
  item_id TEXT NOT NULL REFERENCES items (id),
  buyer_id TEXT NOT NULL REFERENCES users (id),
  seller_id TEXT NOT NULL REFERENCES users (id),
  -- 金額（購入希望の時点の商品価格。円の整数。PAY-2）
  price INTEGER NOT NULL CHECK (price > 0),
  -- 購入者が購入希望のときに選んだ支払い方法（ADR 0021）
  payment_method TEXT NOT NULL CHECK (payment_method IN ('card', 'paypay')),
  status TEXT NOT NULL DEFAULT 'requested'
    CHECK (status IN ('requested', 'rejected', 'expired', 'in_deal', 'cancelled', 'completed')),
  -- 承認期限（購入希望から48時間。DEAL-1）
  approval_deadline TEXT NOT NULL,
  -- 適用した手数料率（ベーシスポイント）・手数料・出品者の売上。取引成立（承認）の時点で記録する（PAY-2, ADR 0003）
  fee_rate_bp INTEGER NULL CHECK (fee_rate_bp BETWEEN 0 AND 10000),
  fee INTEGER NULL CHECK (fee >= 0),
  seller_proceeds INTEGER NULL CHECK (seller_proceeds >= 0),
  -- 学籍番号を開示した日時（承認時。DEAL-1）
  identity_disclosed_at TEXT NULL,
  -- 出品者が確認モーダルに同意した日時（DEAL-2）
  seller_acknowledged_at TEXT NULL,
  -- キャンセルへの同意日時。双方がそろったらキャンセルする（DEAL-4）
  buyer_cancel_agreed_at TEXT NULL,
  seller_cancel_agreed_at TEXT NULL,
  -- 受け渡しが完了した日時と完了方法（DEAL-3, DEAL-5）
  completed_at TEXT NULL,
  handover_method TEXT NULL CHECK (handover_method IN ('qr', 'link', 'code', 'auto')),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  CHECK (buyer_id <> seller_id),
  -- 手数料の3列はそろって入る
  CHECK ((fee_rate_bp IS NULL) = (fee IS NULL) AND (fee IS NULL) = (seller_proceeds IS NULL))
);

-- 商品ごとの購入希望・取引（DEAL-1 の購入希望一覧）
CREATE INDEX transactions_item_status_idx ON transactions (item_id, status);

-- 購入者・出品者ごとの取引一覧（DEAL-1、AUTH-3 マイページ）
CREATE INDEX transactions_buyer_created_at_idx ON transactions (buyer_id, created_at);
CREATE INDEX transactions_seller_created_at_idx ON transactions (seller_id, created_at);

-- 短周期ジョブの期限切れ処理（JOBS-1）
CREATE INDEX transactions_status_approval_deadline_idx ON transactions (status, approval_deadline);

-- 同じ購入者が同じ商品に出せる購入希望は同時に1件だけ
CREATE UNIQUE INDEX transactions_item_buyer_requested_uidx ON transactions (item_id, buyer_id) WHERE status = 'requested';

-- 1つの商品で成立する取引は1件だけ（同じ商品を2人に承認しない。DB）
CREATE UNIQUE INDEX transactions_item_in_deal_uidx ON transactions (item_id) WHERE status IN ('in_deal', 'completed');
