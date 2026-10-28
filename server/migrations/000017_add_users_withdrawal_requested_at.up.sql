-- 退学を申請した日時（AUTH-3, ADMIN, ADR 0034）。本人がアプリから申請し、管理者が削除するまで残る
ALTER TABLE users ADD COLUMN withdrawal_requested_at TEXT NULL;

-- 管理画面で申請中の人を数えて絞り込む
CREATE INDEX users_withdrawal_requested_idx ON users (withdrawal_requested_at) WHERE withdrawal_requested_at IS NOT NULL;
