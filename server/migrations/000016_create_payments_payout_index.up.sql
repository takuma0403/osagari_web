-- 出金（payout）は取引ごとに1行だけ（PAY, ADR 0029）。同時に出金を申請しても二重に記録しない
CREATE UNIQUE INDEX payments_transaction_payout_uidx ON payments (transaction_id) WHERE kind = 'payout';
