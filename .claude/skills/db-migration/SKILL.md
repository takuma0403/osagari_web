---
name: db-migration
description: golang-migrate の SQL マイグレーションでスキーマを変更する手順と規約（AutoMigrate禁止、データモデルとの整合、条件付きUPDATE向けの設計）。テーブル・列・インデックスを追加・変更するときに必ず使う。
argument-hint: "<変更内容 例: transactions テーブルの追加>"
---

# db-migration：スキーマを変更する

## 原則（[docs/tech/database.md](../../../docs/tech/database.md) の DB 章）

- スキーマの変更は `server/migrations/` の SQL ファイルだけで行う。**GORM の AutoMigrate は使わない**（テストを含む）。
- DB は Turso（SQLite互換）。SQLite で動く SQL だけを書く（`ALTER TABLE` の制約に注意。列の削除・型変更はテーブルを作り直す）。

## 手順

1. [docs/product/09-data-model.md](../../../docs/product/09-data-model.md) と [docs/glossary.md](../../../docs/glossary.md) で、テーブル名・列名・列挙値を確認する。データモデルにない項目を足すときは、先に `spec-update` スキルで docs を更新するか、ユーザーに確認する。
2. ファイルを作る：`server/migrations/NNNNNN_<snake_case の説明>.up.sql` と `.down.sql`（6桁の連番。`migrate create -ext sql -dir migrations -seq <name>` で作ってもよい）。
3. 書き方の規約：
   - テーブル名は複数形の snake_case。主キーは `id TEXT PRIMARY KEY`（ULID などの推測しにくいID）。外部に出すIDは連番にしない
   - 日時は `TEXT`（RFC 3339 / UTC）で統一する。`created_at`・`updated_at` を付ける
   - 金額は `INTEGER`（円）、手数料率は `INTEGER`（ベーシスポイント）
   - ステータスは `TEXT NOT NULL` に `CHECK (status IN (...))` を付け、値は glossary に揃える
   - 外部キーは `REFERENCES` を書く（接続時に `PRAGMA foreign_keys = ON` を設定する）
   - 決済まわり（`payments` など）には外部サービスのIDを入れる列（`external_id TEXT NULL`）を最初から作っておく（PAY, DATA）
   - 学生証は `delete_after`（失効・削除日 + 3年）を持たせ、日次ジョブで削除できるようにする（CARD）
   - よく使う絞り込み（status、seller_id、created_at の新着順など）にインデックスを張る
   - トークン・コード・パスワードはハッシュの列だけを持ち、平文の列を作らない
4. `.down.sql` で元に戻せることを確認する（up → down → up）。
5. repository の GORM モデルを、マイグレーションと一致するように更新する。
6. 条件付きUPDATEを使う遷移（承認・完了・キャンセル）は、repository のメソッドとして `WHERE id = ? AND status = ?` を書き、更新件数を返す。

## 確認

- `go test ./...` で、一時ファイルの SQLite にマイグレーションを当てるテストが通ること。
- libSQL（Turso）での動作が未検証の構文を使った場合は、そのことを報告する（tech/open-questions.md）。
