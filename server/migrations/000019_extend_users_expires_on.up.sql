-- 有効期限を在学年限（修業年限×2）に延ばす（AUTH-3, ADR 0047）。
-- 有効・停止のユーザーのうち、有効期限が旧ルールの初期値（入学年度＋修業年限。医学部医学科 211 は6年、それ以外は4年）のままの行だけを直す。
-- 管理者が変えた有効期限と、失効・削除済みのユーザーは変えない。updated_at は repository の timeLayout（マイクロ秒までの固定幅）に揃える
UPDATE users
SET expires_on = printf('%04d-03-31', admission_year + CASE department_code WHEN '211' THEN 12 ELSE 8 END),
    updated_at = strftime('%Y-%m-%dT%H:%M:%S', 'now') || '.000000Z'
WHERE status IN ('active', 'suspended')
  AND expires_on = printf('%04d-03-31', admission_year + CASE department_code WHEN '211' THEN 6 ELSE 4 END);
