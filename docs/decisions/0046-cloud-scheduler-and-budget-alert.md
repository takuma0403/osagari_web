# ADR 0046: Cloud Scheduler のジョブの設定と予算アラート、DB バックアップの切り出し

- 日付: 2026-11-02
- 状態: 採用
- 関連する章ID: JOBS-1, COST, DB

## 背景

#40 で定期処理を本番で動かし、費用の監視とバックアップを整えるにあたり、次のことが決まっていなかった。

- 短周期ジョブ・日次ジョブ（[JOBS-1](../tech/jobs-payment-mock.md)）を Cloud Scheduler でいつ・どう呼ぶか（日次は「毎日深夜」とだけ決まっていた）。
- 予算アラートの金額（[COST](../tech/00-overview.md) では「例：月1,000円」）。
- DB のバックアップ（[DB](../tech/database.md)）の方式。[ADR 0033](0033-account-expiry-and-daily-job.md) で日次ジョブには入れず別に扱うことにしたままだった。

## 決定

- **Cloud Scheduler**：ジョブは2つ（`osagari-jobs-short`・`osagari-jobs-daily`。無料枠は請求先アカウントあたり3ジョブで、同じ請求先のほかのプロジェクトのジョブも合算される）で、Cloud Run と同じ asia-northeast1 に置く。時刻は Asia/Tokyo で、短周期は15分ごと（`*/15 * * * *`）、日次は毎日3時（`0 3 * * *`。有効期限は日本時間の日の終わりなので0時を過ぎていればよく、利用の少ない時間にする）。
- 呼び出しには Cloud Scheduler のサービスアカウント `osagari-scheduler@…` の OIDC ID トークンを付け、audience は Cloud Run のサービスURL（API の `JOBS_OIDC_AUDIENCE` と同じ値）にする。API は公開のサービスなので、このサービスアカウントにロールは付けない（[ADR 0027](0027-short-job-endpoint.md)）。
- 失敗（500 や応答なし）は1分以上あけて3回まで再試行し、1回の期限は300秒（Cloud Run のリクエストの既定の期限）にする。どの処理も何度行っても結果は同じなので、再試行してよい（JOBS-1）。
- **予算アラート**：請求先アカウント全体に月500円の予算をかけ、50%・90%・100%で請求先アカウントの管理者と運用通知のチャンネル（`osagari 運用通知`）に通知する。
- **DB バックアップ**：#40 から外し、#118 で方式を決めて実装する。保持は30日の予定。

## 理由と検討した選択肢

| 選択肢 | 良い点 | 悪い点 |
| --- | --- | --- |
| 予算 月500円（採用） | 無料枠の想定で0円なので、少しの課金でも早く気づける | 小さな課金でも通知が来る |
| 予算 月1,000円（COST の例） | 通知が少ない | 気づくのが遅れる |
| バックアップを #40 で一緒に作る | issue が1つで済む | 方式（同じイメージの Cloud Run ジョブ／日次ジョブに入れる／GitHub Actions）が決まっておらず、server のコードも要る |
| バックアップを別の issue にする（採用） | Scheduler と予算を先に動かせる。方式を落ち着いて決められる | 決まるまでは Turso の1日分の復元だけになる |

## 影響

- [tech/00-overview.md](../tech/00-overview.md) COST、[tech/jobs-payment-mock.md](../tech/jobs-payment-mock.md) JOBS-1、[tech/database.md](../tech/database.md) DB を更新した。
- インフラ：[infra/setup.sh](../../infra/setup.sh) に Cloud Scheduler のサービスアカウントとジョブ、予算を追加した。サービスURLの組み立ては [infra/config.sh](../../infra/config.sh) の `service_url` にまとめ、[infra/deploy-server.sh](../../infra/deploy-server.sh) と共有する。手順は [infra/README.md](../../infra/README.md)。
- 実装：API の変更はない。
