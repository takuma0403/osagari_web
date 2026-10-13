---
name: spec-update
description: 要件・技術方針・デザイントークン・用語を変更する、または未決事項が決まったときに、分割済みの docs を矛盾なく更新し ADR を残す手順。「仕様を変えたい」「〇〇に決めた」「未決事項の〇〇は△△で」と言われたとき、実装中に仕様との違いが出たときに使う。
argument-hint: "<変更内容>"
---

# spec-update：仕様を更新する

## 原則

- 正本は `docs/product/`・`docs/tech/`・`docs/design/README.md`・`docs/api/openapi.yaml`・`docs/glossary.md`。**`docs/_source/` は書き換えない**（分割前の凍結コピー）。
- 1つの事実は1か所に書き、ほかからはリンクする。同じ内容を複数のファイルに書き写さない。
- ユーザーが決めていないことを、仕様として書かない。

## 手順

1. 影響範囲を調べる：変更するキーワードと関係する章IDで `docs/`（`_source` を除く）、`CLAUDE.md`、`app/CLAUDE.md`、`server/CLAUDE.md`、`.claude/` を grep する。
2. 該当する節を書き換える。
   - **章IDは振り直さない。** 節を追加するときは、その接頭辞の続きの番号を使う（DEAL-5 の次は DEAL-6）。節を削除したら、その番号は欠番のままにする
   - 要件と技術の両方に関わるときは両方を直す
   - 数字（期限・回数・金額）を変えたら、design（文言）・glossary・CLAUDE.md の「守るべき決まり」にも同じ数字がないか確かめる
3. 未決事項が決まった場合：`open-questions.md` の該当項目を `- [x]` にして、決定内容と ADR へのリンクを1行で添える。
4. 判断を伴う変更（未決事項の決定、方針の変更、決まりの変更）は ADR を作る：`docs/decisions/NNNN-<kebab-case>.md`（[0000-template.md](../../../docs/decisions/0000-template.md) をコピー）。誤字の修正や言い回しの調整なら不要。
5. `docs/README.md` の対応表と、ルート `CLAUDE.md` の「守るべき決まり」に影響があれば更新する。
6. API やスキーマに影響があれば、`api-contract`・`db-migration` スキルの作業が必要なことを報告する（ユーザーに頼まれていなければ、実装までは進めない）。
7. `node scripts/check-docs.mjs` を実行し、通ることを確認する。

## 報告

- 変更したファイルと章ID
- 作成した ADR
- 実装への影響（直す必要があるコード）
