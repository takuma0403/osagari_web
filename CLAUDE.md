# 佐賀大お下がりフリマ

佐賀大学の学生（`@edu.cc.saga-u.ac.jp`）だけが使えるお下がり品のフリマWebアプリ。教科書が中心。アプリが担うのは出品・一覧・検索・代金のやり取り（フェーズ1はモック）で、連絡と受け渡しは当事者同士が Teams とキャンパスで行う。
構成：`app/`（Vite + React + TS の SPA → Firebase Hosting）と `server/`（Go + Echo → Cloud Run、DBは Turso）。

## 作業の始め方（必ず守る）

1. **[docs/README.md](docs/README.md) を読み**、対応表から作業に関係する章ID・ファイルを特定する。迷ったら `spec-context` スキルを使う。
2. 関係する `docs/product/`（要件）・`docs/tech/`（技術）・`docs/design/`（画面）**だけ**を読む。`docs/_source/` は読まない（分割前の凍結コピー）。
3. **作業が [product/open-questions.md](docs/product/open-questions.md) か [tech/open-questions.md](docs/tech/open-questions.md) の項目に関わるなら、勝手に決めずに止めてユーザーに質問する。**
4. 仕様にない判断が必要になったら、推測で実装せずに確認する。決まったことは `spec-update` スキルで docs に反映する。
5. コード・コミット・PRで仕様を指すときは章ID（例 `DEAL-3`）を使う。

## 守るべき決まり

違反は不具合として扱う。変えるときはユーザーの承認と ADR が必要。

| 決まり | 出典 |
| --- | --- |
| フェーズ1の決済はモック。実際のお金を動かす処理・外部決済APIの呼び出しを書かない | OVERVIEW, PAY |
| チャット・商品Q&A・配送・受け渡し代行は作らない | OVERVIEW |
| 登録できるのは `^\d{8}@edu\.cc\.saga-u\.ac\.jp$` のみ（サーバー側で検証） | AUTHN-2 |
| 登録済みかどうかを画面・応答から判別できないようにする | AUTHN-2 |
| パスワードは argon2id・10文字以上。認証コードはハッシュで保存 | AUTHN-2, AUTHN-5 |
| セッションCookie名は `__session`（HttpOnly, Secure, SameSite=Lax）。ほかのCookieは使わない。端末識別子は localStorage → ヘッダー | API, AUTHN-3, AUTHN-5 |
| 学籍番号は出品者が承認するまで相手に開示しない | DEAL-1 |
| 代金を動かすきっかけ（受け渡し完了）は**購入者の操作**だけ（自動完了は未決） | DEAL-3, DEAL-5 |
| QR・完了リンク・完了コードには取引IDと使い捨てトークンだけを入れ、個人情報・金額を入れない。一度使ったら無効 | DEAL-3 |
| 状態遷移（承認・完了・キャンセルなど）は「現在のステータスが想定どおりのときだけ更新する」条件付きUPDATEで行う | DB |
| 取引成立後のキャンセルは双方の同意のみ。受け渡し完了後はキャンセル・返金不可 | DEAL-4 |
| GORM の AutoMigrate を使わない。スキーマ変更は golang-migrate のSQLファイルだけ | DB |
| 学生証画像は非公開バケット。署名付きURLは管理者APIだけが発行し、閲覧を記録する | CARD, STORAGE |
| 画像はブラウザから署名付きURLで直接アップロード（APIを経由しない） | ARCH, STORAGE |
| 金額は円の整数、手数料率はベーシスポイントの整数。取引ごとに適用した料率を記録する | PAY-2 |
| API の正本は `docs/api/openapi.yaml`。先に契約を更新し、型は生成する（手書きしない） | API-1 |
| 費用をかけない構成（無料枠）を優先する。有料サービスを足すときは確認する | ARCH, COST |

## リポジトリ構成

```
CLAUDE.md            … このファイル（索引）
docs/                … 仕様（README.md が入口）
app/                 … フロントエンド（規約: app/CLAUDE.md）※雛形は未作成
server/              … バックエンド（規約: server/CLAUDE.md）※雛形は未作成
scripts/             … check-docs.mjs（docs検査）、hooks/
.claude/skills/      … 作業手順（下表）
.claude/agents/      … spec-reviewer（仕様との照合）
```

## スキル

| スキル | 使う場面 |
| --- | --- |
| `spec-context` | どの作業でも最初に。関係する仕様を集めて作業ブリーフを作る |
| `implement-feature` | 機能を実装するとき（仕様確認 → 方針承認 → API → DB → server → app → テスト → docs） |
| `api-contract` | API を追加・変更するとき |
| `db-migration` | テーブル・列を変えるとき |
| `design-to-code` | 画面を実装するとき |
| `design-sync` | Claude Design のキャンバスが更新されたとき |
| `spec-update` | 要件・技術方針・デザイントークンを変えるとき、未決事項が決まったとき |

実装が終わったら `spec-reviewer` サブエージェントで仕様との照合を行う。

## コマンド

| 目的 | コマンド |
| --- | --- |
| docs の検査（リンク切れ・章ID重複） | `node scripts/check-docs.mjs` |
| app / server | 雛形を作った時点で `app/CLAUDE.md`・`server/CLAUDE.md` に追記する |

## 書き方

- 返答・ドキュメント・コードコメント・コミットメッセージは日本語。
- コミットは Gitmoji 形式 `<emoji> <type>: <subject>`（例 `✨ feat: 購入希望の承認APIを追加 (DEAL-1)`）。コミットはユーザーの指示があるときだけ。
- `.env` などの機密ファイルは読まない（`.env.example` のみ可）。秘密情報はログやコマンド出力に出さない。
- 依頼されていない範囲のリファクタリングはしない。大きな実装は着手前に方針を箇条書きで示して承認を得る。
