# server（バックエンド）

> このディレクトリで作業するときの規約。全体の決まりはルートの [CLAUDE.md](../CLAUDE.md)、技術方針は [docs/tech/](../docs/tech/00-overview.md)（特に [api.md](../docs/tech/api.md)・[auth.md](../docs/tech/auth.md)・[database.md](../docs/tech/database.md)・[jobs-payment-mock.md](../docs/tech/jobs-payment-mock.md)）。
> **雛形は未作成。** 最初の作業で下の構成どおりに作り、「コマンド」の表を実際の内容に更新すること。

## 技術スタック

| 項目 | 採用 | 出典 |
| --- | --- | --- |
| 言語 | Go 1.23 | API |
| Web | Echo | API |
| API層 | oapi-codegen（Echo / strict server）で `docs/api/openapi.yaml` から `internal/api/api.gen.go` を生成 | API-1 |
| DB | Turso（libSQL）。GORM の SQLite ドライバ + libSQL ドライバ | DB |
| マイグレーション | golang-migrate（`migrations/` の SQL）。**AutoMigrate 禁止** | DB |
| パスワード | argon2id | AUTHN-5 |
| メール | Gmail SMTP（587 / STARTTLS） | AUTHN-7 |
| ストレージ | Cloud Storage の署名付きURL | STORAGE |
| ローカルDB | SQLite ファイル（`file:local.db`）で代用し、本番は Turso | — |

GORM と golang-migrate が libSQL で動くかは未検証（[tech/open-questions.md](../docs/tech/open-questions.md)）。最初に検証し、問題があればユーザーに報告する。

## ディレクトリ構成（予定）

```
server/
  cmd/api/main.go           … 起動、依存の組み立て（DI）
  internal/
    api/api.gen.go          … 生成物（編集しない）
    handler/                … 生成インターフェースの実装。入力の変換と応答だけ（ロジックを書かない）
    service/                … 業務ロジック。状態遷移・権限・手数料計算はここ
    repository/             … GORM によるDBアクセス。条件付きUPDATEもここ
    domain/                 … エンティティと列挙値（docs/glossary.md の名前）
    payment/                … PaymentGateway インターフェースと mock 実装
    mail/                   … Mailer インターフェース、SMTP 実装、テスト用の記録実装
    storage/                … 署名付きURL発行のインターフェースと実装
    clock/                  … Clock インターフェース（テストで時刻を固定する）
    auth/                   … セッション、argon2id、認証コード、レート制限
    jobs/                   … 短周期ジョブ・日次ジョブ（内部用エンドポイントから呼ぶ）
  migrations/               … 000001_create_users.up.sql / .down.sql …
```

## 規約

- **層の向きは handler → service → repository の一方向。** handler で DB を触らない。service はインターフェース越しに repository・payment・mail・storage・clock を使う。
- **状態遷移は条件付きUPDATE**：`UPDATE transactions SET status='in_deal' … WHERE id=? AND status='requested'`。更新件数が0なら競合として `409` と `Error{code:"invalid_status_transition"}` を返す（[DB](../docs/tech/database.md)）。
- 複数テーブルにまたがる処理（承認時の他の購入希望の却下＋枠の解放など）はトランザクションで行う。
- **決済は `payment.Gateway` インターフェース経由のみ**。フェーズ1は mock 実装を差し込み、`payments` に記録するだけ。外部IDの列は空で残す（[JOBS-2](../docs/tech/jobs-payment-mock.md)）。
- 時刻は必ず `clock.Clock` から取る（`time.Now()` を直接呼ばない）。期限（48時間、10分、15分、5分、24時間、30日）の判定をテストできるようにするため。
- 金額は `int64` の円、手数料率はベーシスポイントの `int`。浮動小数点を使わない。端数の扱いは実装時に確認する。
- 学籍番号・メールアドレス・トークン・パスワード・コードをログに出さない。エラー応答にも内部情報を出さない。
- 認証コードとトークンは `crypto/rand` で生成し、ハッシュで保存して定数時間で比較する。
- 権限チェック（その取引の購入者か、出品者か、管理者か）は service で行い、テストを書く。
- 内部用エンドポイント（定期処理）は Cloud Scheduler のサービスアカウントからの呼び出しだけを受け付ける（[JOBS-1](../docs/tech/jobs-payment-mock.md)）。
- スキーマ変更は `db-migration` スキル、API の追加・変更は `api-contract` スキルに従う。

## テスト

- service 層は表駆動テストで、状態遷移ごとに「成功」「想定外のステータス」「権限なし」「期限切れ」を確認する。
- repository 層は一時ファイルの SQLite にマイグレーションを当てて実行する。
- 二重処理の防止（同時に2回承認・完了）は goroutine で並行実行するテストを書く。

## コマンド（雛形作成後に確定させる）

| 目的 | コマンド |
| --- | --- |
| 開発サーバー | `go run ./cmd/api`（:8080） |
| テスト | `go test ./...` |
| Lint | `go vet ./...`（必要なら staticcheck） |
| フォーマット | `gofmt -w .`（Hook で自動実行） |
| APIコード生成 | `go generate ./internal/api` |
| マイグレーション | `migrate -path migrations -database "$DATABASE_URL" up` |
