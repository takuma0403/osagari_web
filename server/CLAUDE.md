# server（バックエンド）

> このディレクトリで作業するときの規約。全体の決まりはルートの [CLAUDE.md](../CLAUDE.md)、技術方針は [docs/tech/](../docs/tech/00-overview.md)（特に [api.md](../docs/tech/api.md)・[auth.md](../docs/tech/auth.md)・[database.md](../docs/tech/database.md)・[jobs-payment-mock.md](../docs/tech/jobs-payment-mock.md)）。
> ローカル環境と libSQL まわりの選定理由は [ADR 0002](../docs/decisions/0002-dev-environment.md)。

## 技術スタック

| 項目 | 採用 | 出典 |
| --- | --- | --- |
| 言語 | Go 1.27（cgo 有効。go-libsql が必要とする） | API, ADR 0002 |
| Web | Echo v5 | API |
| API層 | oapi-codegen（`echo5-server` + strict server、`go tool` で実行）で `docs/api/openapi.yaml` から `internal/api/api.gen.go` を生成 | API-1 |
| DB | Turso（libSQL）。ドライバは go-libsql、GORM の方言は gormlite（`OpenDB` に接続を渡す）。`gorm.io/driver/sqlite`・glebarez/sqlite はシンボル／ドライバ名が衝突するので使わない | DB, ADR 0002 |
| マイグレーション | golang-migrate（`migrations/` の SQL）。`internal/migration` が文ごとに分けて実行する。**AutoMigrate 禁止** | DB |
| パスワード | argon2id | AUTHN-5 |
| メール | Gmail SMTP（587 / STARTTLS）。ローカルは Mailpit（`mail:1025`、受信箱 http://localhost:8025） | AUTHN-7 |
| ストレージ | Cloud Storage の署名付きURL。ローカルは fake-gcs-server（`STORAGE_EMULATOR_HOST`） | STORAGE |
| ローカルDB | compose の libsql-server（Turso 互換。ホストからは `http://localhost:8081`） | ADR 0002 |

GORM と golang-migrate の libSQL での動作は検証済み（`internal/db/libsql_integration_test.go`）。注意点：

- go-libsql は **1回の Exec に複数の文を渡すと2文目以降を黙って捨てる**。アプリのコードでは1回に1文だけを実行する（マイグレーションは `internal/migration` が分割する）。
- リモート接続はプールに残さない（`db.Open` で `SetMaxIdleConns(0)`）。sqld・Turso が約10秒使われなかったストリームを捨て、go-libsql はそれを壊れた接続として報告しないため（[ADR 0012](../docs/decisions/0012-libsql-idle-connections.md)）。
- 一意制約違反は `gorm.ErrDuplicatedKey` に変換されない。判定が必要なら repository 層でエラー文字列（`UNIQUE constraint failed`）を見る。

## ディレクトリ構成

```
server/
  compose.yaml              … ローカル環境（db / storage / storage-init / mail / migrate / api）
  Dockerfile                … dev（air）/ build / prod（distroless/cc。本番へのデプロイは ../infra/README.md）
  .air.toml                 … ホットリロードの設定
  .env.example              … 環境変数の一覧
  dev/                      … ローカル専用のスクリプト（バケット作成など）
  cmd/api/main.go           … 起動、依存の組み立て（DI）
  cmd/migrate/main.go       … マイグレーションの実行（up / down / version）
  internal/
    api/api.gen.go          … 生成物（編集しない）。設定は api/oapi-codegen.yaml
    db/                     … libSQL の接続と GORM の初期化
    migration/              … golang-migrate を libSQL で動かす（文の分割）
    handler/                … 生成インターフェースの実装。入力の変換と応答だけ（ロジックを書かない）
    service/                … 業務ロジック。状態遷移・権限・手数料計算はここ
    repository/             … GORM によるDBアクセス。条件付きUPDATEもここ
    domain/                 … エンティティと列挙値（docs/glossary.md の名前）
    apperr/                 … 共通エラー（code・status・message）。service が返し、server が Error 応答に変換する
    payment/                … PaymentGateway インターフェースと mock 実装
    mail/                   … Mailer インターフェース、SMTP 実装、テスト用の記録実装
    storage/                … 署名付きURL発行のインターフェースと実装
    clock/                  … Clock インターフェース（テストで時刻を固定する。clock.NewFake）
    logging/                … JSON ロガー（個人情報・秘密情報を伏せ字にする）
    server/                 … Echo の組み立て（リクエストID・アクセスログ・panic 回復・エラー応答・ルート登録・`__session` の検証）
    testutil/testdb/        … repository テスト用の一時 SQLite（migrations/ を適用済み）
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
- 金額は `int64` の円、手数料率はベーシスポイントの `int`。浮動小数点を使わない。手数料の1円未満は切り捨てる（`domain.CalcFee`。PAY-2, ADR 0017）。
- 学籍番号・メールアドレス・トークン・パスワード・コードをログに出さない（`logging` が伏せ字にするが、それに頼らず渡さない）。エラー応答にも内部情報を出さない。
- エラーは `apperr` で返す（`apperr.InvalidStatusTransition("…")`、想定外は `apperr.Internal(err)`）。handler は service のエラーをそのまま return し、応答の形は `internal/server` が揃える。
- 認証コードとトークンは `crypto/rand` で生成し、ハッシュで保存して定数時間で比較する。
- 権限チェック（その取引の購入者か、出品者か、管理者か）は service で行い、テストを書く。
- 内部用エンドポイント（定期処理）は Cloud Scheduler のサービスアカウントからの呼び出しだけを受け付ける（[JOBS-1](../docs/tech/jobs-payment-mock.md)）。
- スキーマ変更は `db-migration` スキル、API の追加・変更は `api-contract` スキルに従う。

## テスト

- service 層は表駆動テストで、状態遷移ごとに「成功」「想定外のステータス」「権限なし」「期限切れ」を確認する。
- repository 層は一時ファイルの SQLite にマイグレーションを当てて実行する。`gdb := testdb.New(t)`（`internal/testutil/testdb`）でテストごとに新しい DB が用意される。
- 二重処理の防止（同時に2回承認・完了）は goroutine で並行実行するテストを書く。

## コマンド

`server/` で実行する。Go のコマンドはコンテナ内で動かす（ホストに Go 1.27 があれば直接でもよい）。

| 目的 | コマンド |
| --- | --- |
| 起動（全サービス） | `docker compose up -d`（api は air でホットリロード、:8080） |
| ログ | `docker compose logs -f api` |
| 停止 / データも消す | `docker compose down` / `docker compose down -v` |
| テスト | `docker compose run --rm api go test ./...` |
| libSQL の統合テスト | `docker compose run --rm -e TEST_DATABASE_URL=http://db:8080 api go test ./internal/db/...`（未設定ならスキップ） |
| ストレージの統合テスト | `docker compose run --rm -e TEST_STORAGE_EMULATOR_HOST=storage:4443 api go test ./internal/storage/...`（未設定ならスキップ） |
| Lint | `docker compose run --rm api go vet ./...` |
| フォーマット | `gofmt -w .`（Hook で自動実行） |
| APIコード生成 | `docker compose run --rm api go generate ./internal/api` |
| マイグレーション | `docker compose run --rm migrate`（up）／ `docker compose run --rm api go run ./cmd/migrate down` |
