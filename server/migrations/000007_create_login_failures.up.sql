-- パスワードの入力ミスの記録（AUTHN-6, ADR 0011）。10回続いたら15分間ログインを止める。
-- 未登録・削除済みのアドレスも同じように数えて止めるため（登録の有無を判別させない。AUTHN-2）、ユーザーではなく正規化したメールアドレスで持つ
CREATE TABLE login_failures (
  email TEXT PRIMARY KEY,
  -- 照合の前に数える入力の回数。パスワードが一致したら0に戻す
  failed_attempts INTEGER NOT NULL DEFAULT 0 CHECK (failed_attempts >= 0),
  -- この日時までログインを止める。止めていなければ NULL
  locked_until TEXT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

-- コード送信の回数制限（AUTHN-6）。同じアドレスへの送信を用途をまとめて数える
CREATE INDEX verification_codes_email_created_at_idx ON verification_codes (email, created_at);

-- 全体の送信数（AUTHN-6）。直近24時間に送ったコードの数を数える
CREATE INDEX verification_codes_created_at_idx ON verification_codes (created_at);
