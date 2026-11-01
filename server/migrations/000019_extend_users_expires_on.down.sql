-- 有効期限を旧ルールの初期値（入学年度＋修業年限）に戻す。新しい初期値のままの行だけを戻す
UPDATE users
SET expires_on = printf('%04d-03-31', admission_year + CASE department_code WHEN '211' THEN 6 ELSE 4 END),
    updated_at = strftime('%Y-%m-%dT%H:%M:%S', 'now') || '.000000Z'
WHERE status IN ('active', 'suspended')
  AND expires_on = printf('%04d-03-31', admission_year + CASE department_code WHEN '211' THEN 12 ELSE 8 END);
