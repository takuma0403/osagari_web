# ADR 0007: 新規登録の一時トークン・登録有無の隠し方・推定API

- 日付: 2026-10-14
- 状態: 採用
- 関連する章ID: AUTHN-2, AUTHN-4, AUTHN-6, AUTH-2, AUTH-4, API-1

## 背景

#9（新規登録API）を実装するにあたり、仕様に書かれていない次の点を決める必要があった。

- [ADR 0006](0006-auth-secret-hashing.md) で「#9 で決める」とした、パスワード設定用の一時トークン（AUTHN-2、15分有効）の保存先。
- AUTHN-2 は「登録済みのアドレスにはコードの代わりに案内を送り、画面上はどちらも同じ表示にする」とだけ書いている。応答だけでなく、所要時間や、その後のコード確認の応答からも登録の有無が分からないようにする方法が決まっていない。
- SignUp 画面は入力中に「理工学部 理工学科・2026年度入学（有効期限 2030年3月31日）」を表示するが、どこで推定するかが決まっていない。
- AUTHN-6 は「コードの入力ミスは5回まで」としているが、同時に大量の入力を送られたときに上限を守る方法が決まっていない。

あわせて、未決事項のうち次の3つは**決めずに**、現在の仕様の暫定の扱いで実装した（[open-questions](../product/open-questions.md) は未決のまま）。

- 学部・学科コードの最新版と大学院の研究科コード → AUTH-4 の表の10コードだけを使い、表にないコードは学部を不明（`department` が NULL）、有効期限は入学年度＋4年にする（[ADR 0047](0047-account-period-enrollment-limit.md) で＋8年に変更）。
- 教職員・留学生の扱い → `^\d{8}@edu\.cc\.saga-u\.ac\.jp$` だけを受け付ける。
- 有効期限の延長 → 初期値の計算だけを実装する。

## 決定

- 一時トークンは新しいテーブル `password_setup_tokens`（email, purpose = `signup` / `password_reset`, token_hash, expires_at, consumed_at）に保存する。値は256ビットの乱数で、DB には SHA-256 だけを持つ。`consumed_at IS NULL AND expires_at > now` を条件にした UPDATE で使用済みにし、一度だけ使えるようにする。パスワード再設定（AUTHN-4）でも同じテーブルを使う。
- コード送信では、**登録済みのアドレスでも未登録と同じ処理**（コードの生成・argon2id・`verification_codes` への保存・メール1通の送信）を行い、メールの本文だけを変える（登録済みにはコードを書かず案内を送る）。応答・DB への書き込み・計算量・送信が同じなので、所要時間も揃う。登録済みのアドレスのコードは誰にも届かないので確認には使えず、コード確認の応答（入力ミスと残り回数）も未登録のときと同じになる。万一コードを当てても、アカウント作成で一意制約に当たり、無効なトークンと同じ応答にする。
- 学部・入学年度・有効期限の推定は、登録状態に触れない公開の API `POST /auth/signup/estimate` で返す。学部の表はサーバーの1か所（`server/internal/domain/department.go`）だけに置く。メールアドレスを URL に載せてリクエストログに残さないため、GET のクエリではなく POST の本文で受け取る。
- コードの入力回数は**照合の前に**数える（`failed_attempts < 5` を条件に +1 する UPDATE）。数えられなかったら照合しない。成功した回も数に入るが、成功したコードは使用済みになるので影響しない。
- 機能固有のエラーコード `invalid_email`・`invalid_verification_code`・`verification_code_expired`・`invalid_setup_token`・`password_too_short` を追加する（いずれも 400）。`password_too_short` は [ADR 0008](0008-password-max-length.md) で `invalid_password_length` に変えた。

## 理由と検討した選択肢

| 選択肢 | 良い点 | 悪い点 |
| --- | --- | --- |
| **一時トークンを新しいテーブルに保存（採用）** | コードとトークンの寿命を別々に持てる。再設定でも使える | テーブルが1つ増える |
| `verification_codes` に列を足す | テーブルが増えない | 1行にコードとトークンの寿命が混ざる |
| 署名付きトークン（DB なし） | 保存が要らない | 署名鍵の管理が要る。一度だけ使える保証には結局 DB が要る |
| 登録済みならコードを保存しない | 書き込みが減る | 所要時間と、コード確認の応答（行がないので「期限切れ」になる）から登録の有無が分かる |
| **登録済みでもコードを保存する（採用）** | 応答・時間・その後の確認まで同じ経路になる | 使われないコードの行ができる |
| app 側にも学部の表を持つ | API 呼び出しが要らない | 表を二重に管理することになる |
| **推定 API を追加（採用）** | 表が1か所。登録の有無は漏れない | API が1本増える |
| 照合の後に入力ミスを数える | ミスだけを数えられる | 同時に大量に送られると、数え終わる前に何度も照合でき、5回の上限を超えて総当たりできる |
| **照合の前に数える（採用）** | 同時に送られても照合は1つのコードにつき5回まで | 成功した回も数に入る（使用済みになるので問題ない） |

## 影響

- 更新したドキュメント：[tech/auth.md](../tech/auth.md)（AUTHN-2, AUTHN-6）、[tech/api.md](../tech/api.md)（API-1 のエラーコード）、[glossary.md](../glossary.md)、[api/openapi.yaml](../api/openapi.yaml)
- 実装：`server/migrations/000005_create_password_setup_tokens`、`server/internal/domain`（`department.go`・`student.go`）、`server/internal/repository`、`server/internal/service/signup.go`、`server/internal/handler/auth.go`
- 送信回数の制限（1分に1回・1時間に5回など、AUTHN-6）は #12 で実装する。
