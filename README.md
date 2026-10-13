# 佐賀大生用お下がりフリマアプリ

佐賀大学の学生（`@edu.cc.saga-u.ac.jp`）だけが使える、お下がり品（教科書・家具・家電など）の売買Webアプリです。フェーズ1の決済はモックです。

- フロントエンド：`app/`（Vite + React + TypeScript → Firebase Hosting）
- バックエンド：`server/`（Go + Echo → Cloud Run、DB は Turso）
- どちらも雛形は未作成です。最初の実装依頼で作ります。

## ドキュメント

| 読みたいもの | 場所 |
| --- | --- |
| 仕様の入口・機能ごとの対応表 | [docs/README.md](docs/README.md) |
| 要件定義（機能別） | [docs/product/](docs/product/00-overview.md) |
| 技術構成 | [docs/tech/](docs/tech/00-overview.md) |
| デザイン（トークン・画面） | [docs/design/README.md](docs/design/README.md) ／ [Claude Design キャンバス](https://claude.ai/artifact/QdafhFt85mz3WvYPPtDGEH) |
| API 契約 | [docs/api/openapi.yaml](docs/api/openapi.yaml) |
| 未決事項 | [要件](docs/product/open-questions.md) ／ [技術](docs/tech/open-questions.md) |
| 設計判断の記録 | [docs/decisions/](docs/decisions/0001-docs-structure.md) |

## AIエージェントで開発する

エージェント向けの指示は [CLAUDE.md](CLAUDE.md)（全体）、[app/CLAUDE.md](app/CLAUDE.md)、[server/CLAUDE.md](server/CLAUDE.md) にあります。Claude Code では次のように頼むと、要件・技術・デザインを読んでから作業します。

| やりたいこと | 指示の例 |
| --- | --- |
| 仕様を確認する | `/spec-context 受け渡しQR` |
| 機能を実装する | `/implement-feature 新規登録とログイン` |
| 画面だけ作る | `/design-to-code Main` |
| API を追加する | `/api-contract 商品一覧API` |
| テーブルを変える | `/db-migration items テーブルを追加` |
| デザインを取り込み直す | `/design-sync` |
| 仕様を変える・未決事項を決める | `/spec-update 自動完了は導入しない` |

- 実装後は `spec-reviewer` サブエージェントが、仕様と食い違いがないかを照合します。
- 未決事項に関わる作業では、エージェントは止まって質問します。
- docs を編集すると、Hook が `node scripts/check-docs.mjs`（リンク切れ・章IDの重複の検査）を自動で実行します。

## 開発環境

- Node.js 20、Go 1.23、Firebase CLI、gcloud CLI（雛形を作るときに必要なものを追記します）
