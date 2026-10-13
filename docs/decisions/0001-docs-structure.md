# ADR 0001: ドキュメントの分割とエージェント用ハーネスの導入

- 日付: 2026-10-14
- 状態: 採用
- 関連する章ID: FE-1, API-1

## 背景

- 要件定義書（約23KB）と技術構成書（約16KB）が1ファイルずつにまとまっており、エージェントが作業のたびに全文を読む必要があった。
- 両文書は Claude Docs から書き出したもので、図3点（取引の流れ・技術構成・ログインの流れ）が `[embedded content]` として失われていた。
- app（フロントエンド）と server（バックエンド）を同じリポジトリで作るため、API の食い違いを防ぐ仕組みが必要だった。

## 決定

1. `docs/product/`（要件）と `docs/tech/`（技術）に機能別に分割し、見出しに章IDを付ける。元文書は `docs/_source/` に凍結保存し、`scripts/check-docs.mjs --coverage` で全行が移行済みであることを確認した。
2. 失われた図は本文の記述だけを根拠に Mermaid で復元し、「本文から復元」と明記する。
3. API 契約の正本を `docs/api/openapi.yaml` とし、Go（oapi-codegen / Echo strict server）と TS（openapi-typescript）の型を生成する。
4. フロントのスタイリングは CSS Modules + CSS変数。トークンは `docs/design/README.md` を正とする。
5. CLAUDE.md（ルート・app・server）、Skills、spec-reviewer サブエージェント、PostToolUse Hook でエージェントの作業手順を固定する。

## 理由と検討した選択肢

| 選択肢 | 良い点 | 悪い点 |
| --- | --- | --- |
| 原本を保ったまま索引とIDだけ付ける | 欠落の心配がない | 1ファイルが大きく、読む量が減らない |
| **機能別に分割（採用）** | 必要な部分だけ読める | 分割時の欠落 → coverage 検査で対策 |
| API を手書きで揃える | 導入が軽い | フロントとサーバーを別のエージェントが触ると食い違う |
| **OpenAPI を正本にする（採用）** | 型が両側で一致する | 生成の手間 |

## 影響

- 旧パス `docs/requirements.md` `docs/architecture.md` は廃止（`docs/_source/` へ移動）。
- 追加した決定は tech/frontend.md の FE-1、tech/api.md の API-1 に記載。
- 分割時に見つかった確認事項は product/open-questions.md の OPEN-DERIVED に記載。
