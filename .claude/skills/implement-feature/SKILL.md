---
name: implement-feature
description: 要件定義に沿って機能を端から端まで実装する手順（仕様確認 → 方針の承認 → OpenAPI → マイグレーション → server → app → テスト → docs更新 → 仕様照合）。「〇〇機能を実装して」「〇〇画面を作って」「〇〇APIを作って」と頼まれたときに使う。
argument-hint: "<機能名 例: 購入希望と承認>"
---

# implement-feature：機能を実装する

対象: `$ARGUMENTS`

## 0. 前提の確認

- `app/` または `server/` の雛形がまだない場合は、雛形作りから始める。その場合も、まず方針を示してユーザーの承認を得る。構成は `app/CLAUDE.md` と `server/CLAUDE.md` の「ディレクトリ構成（予定）」に従う。作ったら各 CLAUDE.md の「コマンド」を実際の内容に更新する。

## 1. 仕様を集める

`spec-context` スキルで作業ブリーフを作る。**「未決事項」か「仕様から読み取れない点」が1つでもあれば、ここで止めてユーザーに質問する。**

## 2. 方針を示して承認を得る

コードを書く前に、次の内容を箇条書きで示し、承認を得る。

- 対象の章ID
- 追加・変更する API（メソッド、パス、主なリクエスト・レスポンス）
- 追加・変更するテーブルと列
- server 側の service に置くロジックと、状態遷移（前のステータス → 後のステータス）
- app 側の画面・ルート・使う共通部品
- テストの観点
- 今回はやらないこと

## 3. 実装する（この順番で）

1. **API契約**：`api-contract` スキルで `docs/api/openapi.yaml` を更新し、両側の型を生成する
2. **DB**：`db-migration` スキルでマイグレーションを追加する
3. **server**：domain → repository → service（テストと一緒に）→ handler
   - 状態遷移は条件付きUPDATE、時刻は Clock、決済は payment.Gateway（mock）
4. **app**：`design-to-code` スキルで画面を作る。API は生成した型と `src/api/client.ts` を使う
5. **テスト**：server は `go test ./...`、app は `npm test` と `npm run typecheck`。すべて通ることを確認する

小さな単位で動作確認しながら進め、最後にまとめて書かない。

## 4. docs を更新する

- 実装の途中で決まったこと、仕様との違いがあれば `spec-update` スキルで docs に反映する（黙って実装だけ変えない）
- 新しい用語を使ったら `docs/glossary.md` に追加する
- `docs/README.md` の対応表に変化があれば更新する
- `node scripts/check-docs.mjs` が通ることを確認する

## 5. 仕様と照合する

`spec-reviewer` サブエージェントに、変更したファイルと対象の章IDを渡してレビューを頼む。指摘を直してから完了を報告する。

## 完了報告に含めるもの

- 対応した issue 番号と、その issue の完了条件を満たしたか
- 実装した章IDと、その章で実装していない部分
- 実行したテストとその結果（失敗があればそのまま書く）
- 更新した docs
- 残っている未決事項・確認事項
- コミットはユーザーの指示があるときだけ（Gitmoji・日本語・章ID付き）
