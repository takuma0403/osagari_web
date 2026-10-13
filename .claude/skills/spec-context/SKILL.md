---
name: spec-context
description: 作業に関係する要件・技術・画面・データ・未決事項を docs から集め、章ID付きの「作業ブリーフ」にまとめる。このリポジトリで機能の実装・修正・調査・見積もりを始める前に必ず使う。「〇〇の仕様は？」「〇〇を作って」「〇〇を直して」など、アプリの機能に触れる依頼すべてが対象。
argument-hint: "<機能やキーワード 例: 受け渡しQR>"
---

# spec-context：作業ブリーフを作る

対象: `$ARGUMENTS`

## 手順

1. [docs/README.md](../../../docs/README.md) の「機能 × 仕様 の対応表」から、対象に該当する行を探す。該当行がなければ、`docs/product` と `docs/tech` をキーワードで grep する（`docs/_source` は除く）。
2. 該当する章を **その節だけ** 読む。読む順番：
   1. `docs/product/` の要件（何を作るか）
   2. `docs/tech/` の技術（どう作るか）
   3. `docs/design/README.md` の DS-4 で画面を特定し、`docs/design/screens/<画面>.dc.html` を読む（文言・状態・バリエーション）
   4. `docs/product/09-data-model.md` と `docs/glossary.md` で関係するエンティティと名前
   5. `docs/api/openapi.yaml` に既存のエンドポイントがあるか
3. **未決事項を照合する**：`docs/product/open-questions.md` と `docs/tech/open-questions.md` を全件読み、対象に関わる項目を挙げる。
4. ルートの `CLAUDE.md` の「守るべき決まり」から、対象に関わる行を挙げる。
5. 既存の実装（`app/`・`server/`）があれば、関係するファイルを探して一覧にする。

## 出力形式（作業ブリーフ）

```markdown
## 作業ブリーフ：<対象>

### 関係する章
- DEAL-3 受け渡し完了の確認（docs/product/05-deal-flow.md）— 要点を1〜2行
- …

### 画面
- DealQR（docs/design/screens/DealQR.dc.html）— 表示要素・状態の要点

### データ・API
- Transaction（status, …）/ HandoverToken
- 既存API: なし（新規）/ GET /api/…

### 守るべき決まり
- QRに個人情報・金額を入れない（DEAL-3）
- …

### 未決事項（⚠ 決めずに確認が必要）
- 自動完了を導入するか（product/open-questions.md）

### 仕様から読み取れない点
- …（推測で埋めずに列挙する）

### 既存の実装
- server/internal/service/deal.go …
```

## 注意

- 要約するときに仕様の数字（48時間、5分、24時間、10回、15分など）を丸めたり省いたりしない。
- 「仕様から読み取れない点」と「未決事項」がある場合、それを解決せずに実装へ進まない。ユーザーに質問する。
- ブリーフは会話に出すだけで、ファイルには保存しない。
