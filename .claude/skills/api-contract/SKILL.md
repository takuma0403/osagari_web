---
name: api-contract
description: docs/api/openapi.yaml（API契約の正本）を先に更新し、Go（oapi-codegen）と TypeScript（openapi-typescript）の型を生成する手順と命名規約。APIエンドポイントやリクエスト・レスポンスの形を追加・変更するときに必ず使う。
argument-hint: "<追加・変更するAPI>"
---

# api-contract：API契約を更新する

## 原則

- **正本は `docs/api/openapi.yaml`。** 実装（Go のハンドラー、TS の型）を先に書かない。
- 生成物（`server/internal/api/api.gen.go`、`app/src/api/schema.gen.ts`）は手で編集しない。

## 手順

1. `docs/glossary.md` で名前を確認する。ない用語は先に追加する。
2. `openapi.yaml` を編集する。
   - パス：`/api` から下の部分を、複数形の名詞で書く（`/items`、`/items/{itemId}`、`/transactions/{transactionId}/approve`）。状態を変える操作は `POST /…/{id}/<動詞>` にする
   - `operationId`：lowerCamelCase の動詞 + 名詞（`listItems`、`approveTransaction`）
   - `tags`：機能単位（auth, items, transactions, handover, student-cards, sales, admin, system, internal）
   - `description` の先頭に章IDを書く（例 `DEAL-3: …`）
   - スキーマ名：UpperCamelCase（glossary の型名）。プロパティ名は lowerCamelCase。列挙値は glossary の値（snake_case）
   - 金額は `integer`（円）、手数料率は `integer`（ベーシスポイント）、日時は `string` / `format: date-time`（UTC）
   - エラーは `components/responses/Error`（`Error{code, message}`）を使う。`code` は snake_case
   - 認証が不要な操作だけ `security: []` を付ける（既定はセッション必須）
   - 管理者向けは `/admin/**`、定期処理は `/internal/**`、教職員向けの認証・身分証の提出は `/staff/**`（AUTHN-8。ログイン後の共通APIは学生と同じパス）
   - 個人情報（学籍番号、教職員のメールアドレス）を返すのは、仕様上開示してよい場面だけ（DEAL-1：承認後の当事者のみ）
3. 型を生成する（雛形を作った後）。
   - server：`cd server && go generate ./internal/api`
   - app：`cd app && npm run gen:api`
4. 生成された型に合わせて handler と app 側を実装し、両方がビルドできることを確認する。
5. 互換性のない変更（項目の削除・名前の変更）をするときは、app と server を同じ変更の中で直す。

## 確認

- 必要なら `npx @redocly/cli lint docs/api/openapi.yaml` で構文を検査する（ネットワークが必要。使う前にユーザーに確認する）。
- 追加した operation が `docs/README.md` の対応表の機能に結びついているか確認する。
