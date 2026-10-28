-- 手数料率を最後に変更した管理者（PAY-2, ADMIN, ADR 0034）。初期値のままなら NULL。変更した日時は updated_at
ALTER TABLE settings ADD COLUMN fee_rate_updated_by TEXT NULL REFERENCES users (id);
