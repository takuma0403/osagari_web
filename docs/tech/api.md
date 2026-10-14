# API（技術）

> 関連: [API契約（OpenAPI）](../api/openapi.yaml) / [認証](auth.md) / [取引フロー（要件）](../product/05-deal-flow.md) / 実装規約: [server/CLAUDE.md](../../server/CLAUDE.md)

## API API

APIは Cloud Run で動かし、Firebase Hosting の rewrites で `/api/**` を転送します。フロントとAPIが同じドメインになるので、CORSやサードパーティCookieの問題を避けられます。

- **言語**：Go を使い、Webフレームワークは Echo にします。
- **リージョン**：東京（asia-northeast1）にします。
- **最小インスタンス数**：0にします。待機中の費用はかかりませんが、しばらくアクセスがないと初回の応答が遅れます。Go は単一のバイナリで起動が速いため、この遅れは小さく抑えられます。
- **無料枠**：Cloud Run には毎月、200万リクエストと18万vCPU秒の無料枠があります（[Cloud Run pricing](https://cloud.google.com/run/pricing)、us-central1基準）。学内アプリの規模なら収まる見込みです。
- **秘密情報**：Gmailのアプリパスワード、Tursoの接続トークン、セッション署名鍵は Secret Manager に保存します。

**Cookieの注意点**：Firebase Hosting から Cloud Run に転送するとき、Cookieは `__session` という名前のもの以外すべて削除されます（[Firebase: Manage cache behavior](https://firebase.google.com/docs/hosting/manage-cache)）。そのため、ログインセッションのCookie名は `__session` にします。

APIが担当する主な処理は次のとおりです。

- 新規登録、パスワードによるログイン、認証コードの発行と照合、パスワードの再設定、セッション管理
- 商品の登録、一覧、検索
- 購入希望、承認、キャンセルなどの取引の状態遷移
- 受け渡しトークン（QR・リンク・コード）の発行と照合
- 画像アップロード用の署名付きURLの発行
- 決済処理（当面はモック）
- 管理者向けの処理（学生証の審査、ユーザーの停止、手数料率の設定など）

**教職員用のAPI（検討保留）**：教職員にも対応する予定で、教職員用の画面とAPIは学生用と分ける前提です（例：`/api/staff/**`）。詳細は要件定義書の「教職員の利用」で検討中です。

> 要件定義書の「教職員の利用」→ [product/01-account-auth.md](../product/01-account-auth.md) の AUTH-5

### API-1 実装方針（ハーネス整備時の決定）

元文書に追加した決定事項です（[ADR 0001](../decisions/0001-docs-structure.md)）。

- **API契約の正本は [docs/api/openapi.yaml](../api/openapi.yaml)**。エンドポイントを追加・変更するときは、先に OpenAPI を更新し、Go 側は oapi-codegen（Echo / strict server）、フロント側は openapi-typescript で型を生成する。
- エラー応答は OpenAPI の `Error` スキーマ（`code` と `message`）に統一する。未実装のパスやルーティングのエラーも同じ形式で返す。`message` は利用者に表示できる日本語だけにし、内部の原因（SQL のエラーなど）は応答に出さずログにだけ残す。
- 共通のエラーコードは次のとおり。機能固有のコードは、その機能を実装するときにこの表へ追加する。

  | code | HTTP | 使う場面 |
  | --- | --- | --- |
  | `bad_request` | 400 | 入力が不正 |
  | `unauthorized` | 401 | ログインしていない |
  | `forbidden` | 403 | 権限がない（取引の当事者でない、管理者でない など） |
  | `not_found` | 404 | 対象がない、未実装のパス |
  | `method_not_allowed` | 405 | パスはあるがメソッドが違う |
  | `invalid_status_transition` | 409 | 条件付きUPDATE の更新件数が0（[DB](database.md)） |
  | `too_many_requests` | 429 | レート制限（[AUTHN](auth.md)） |
  | `internal_error` | 500 | 想定外のエラー |

  機能固有のコード：

  | code | HTTP | 使う場面 |
  | --- | --- | --- |
  | `invalid_email` | 400 | 大学のメールアドレス（`^\d{8}@edu\.cc\.saga-u\.ac\.jp$`）の形式ではない（[AUTHN-2](auth.md)） |
  | `invalid_verification_code` | 400 | 認証コードが違う。メッセージに残りの回数を入れる（[AUTHN-6](auth.md)） |
  | `verification_code_expired` | 400 | 認証コードが期限切れ・使用済み・入力の上限に達した |
  | `invalid_setup_token` | 400 | パスワード設定用の一時トークンが無効・期限切れ・使用済み |
  | `password_too_short` | 400 | パスワードが10文字未満（[AUTHN-5](auth.md)） |

  表にない 4xx（413・415 など）は HTTP ステータスをそのまま使い、`code` は `bad_request` にする。5xx はすべて 500 `internal_error` にする。

- ログは JSON で標準出力に出す（Cloud Logging が `severity` を読む）。学籍番号・メールアドレス・トークン・パスワード・コードは伏せ字にし、アクセスログには実際の URI ではなくルートのパターン（例 `/api/items/:id`）を記録する。各リクエストに `X-Request-Id` を付け、ログに `request_id` として残す。
