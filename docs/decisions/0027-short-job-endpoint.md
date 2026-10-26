# ADR 0027: 短周期ジョブの内部用エンドポイントの認証と、処理の範囲

- 日付: 2026-10-26
- 状態: 採用
- 関連する章ID: JOBS-1, DEAL-1, DEAL-5, AUTHN-6

## 背景

#31 で短周期ジョブ（[JOBS-1](../tech/jobs-payment-mock.md)）を実装するにあたり、次のことが決まっていなかった。

- 「Cloud Scheduler のサービスアカウントからの呼び出しだけを受け付ける」をどう確かめるか。API は Firebase Hosting から誰でも呼べる1つの Cloud Run サービスなので、Cloud Run の IAM（呼び出し元の制限）は使えない。ローカルには Cloud Scheduler がない。
- 「期限切れのログインコード・受け渡しトークンの削除」で、どのテーブルを消すか。
- 未決事項「自動完了を導入するか（[DEAL-5](../product/05-deal-flow.md)）」を、このジョブでどう扱うか。

## 決定

- エンドポイントは `POST /api/internal/jobs/short`。`/api/internal/**` はセッションを使わず、`Authorization: Bearer` のトークンで呼び出し元を確かめる。確かめられなければ、管理APIと同じく存在しないパスと同じ 404 にする。
- 本番は Cloud Scheduler の OIDC ID トークンをアプリで検証する（`google.golang.org/api/idtoken`）。署名・発行者・期限・audience（`JOBS_OIDC_AUDIENCE`）を検証し、`email` が Cloud Scheduler のサービスアカウント（`JOBS_SERVICE_ACCOUNT_EMAIL`）で `email_verified` が true であることを確かめる。
- ローカルは代替として、環境変数 `JOBS_LOCAL_SECRET` の共有シークレットを Bearer トークンとして定数時間で比べる。OIDC の設定とシークレットはどちらか一方だけを設定し、両方・どちらもないときは起動しない。
- 削除するのは、期限切れの `verification_codes`（作ってから24時間は消さない。AUTHN-6）・`login_challenges`・`password_setup_tokens`・`handover_tokens`（使用済みを含む）と、24時間更新のない `login_failures`（ADR 0011）。
- 自動完了（DEAL-5）と催促通知は、導入するか決まるまでジョブに入れない。決まったら別の issue で足す。
- 処理は1つが失敗してもほかを続け、失敗があれば 500 を返して Cloud Scheduler に再試行させる。どの処理も何度行っても結果は同じにする（期限切れは1件ずつ条件付きUPDATE で行い、その間に承認・却下されたものは飛ばす）。

## 理由と検討した選択肢

| 選択肢 | 良い点 | 悪い点 |
| --- | --- | --- |
| OIDC ID トークンをアプリで検証（採用） | Cloud Scheduler の標準の仕組み。秘密の値を管理しなくてよい。追加の費用・サービスがない | Google の公開鍵の取得に外部への通信が要る（ライブラリがキャッシュする） |
| 共有シークレットだけ | 実装が簡単 | 本番でも値の保管と漏れたときの差し替えが要る |
| ジョブ用に別の Cloud Run サービスを立て、IAM で制限 | アプリで検証しなくてよい | サービスとデプロイが増える |

## 影響

- [tech/jobs-payment-mock.md](../tech/jobs-payment-mock.md) JOBS-1：エンドポイント・認証・削除の条件
- [api/openapi.yaml](../api/openapi.yaml)：`runShortJob`、securityScheme `schedulerOidc`
- 実装：`server/internal/jobs`（`Short`・`OIDCVerifier`・`SecretVerifier`）、`internal/server/session.go`（`/api/internal/**` の確認）、`DealService.ExpireRequests`・`DeleteExpiredHandoverTokens`、`AuthCleanupService.DeleteExpiredCodes`、`cmd/api/main.go`、`.env.example`・`compose.yaml`
- デプロイ（M7）：Cloud Scheduler のジョブに OIDC トークン（audience とサービスアカウント）を設定し、Cloud Run に同じ値を環境変数で渡す
