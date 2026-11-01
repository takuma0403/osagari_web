DROP INDEX IF EXISTS users_withdrawal_requested_idx;
ALTER TABLE users DROP COLUMN withdrawal_requested_at;
