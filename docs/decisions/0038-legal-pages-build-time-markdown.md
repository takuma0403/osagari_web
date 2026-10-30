# ADR 0038: 利用規約・プライバシーポリシーの画面は正本の Markdown をビルド時に HTML にして載せる

- 日付: 2026-10-31
- 状態: 採用
- 関連する章ID: AUTH-1, FE-1

## 背景

`/terms`・`/privacy` は #13 で仮ページとして置いていた。本文の正本は [ADR 0036](0036-terms-and-privacy.md) で `docs/legal/terms.md`・`privacy.md` に決まり、画面にはこの本文を載せる（#66）。画面のデザインはない。本文は条文・番号付きの項・表を含み、今後も改定される。

## 決定

- 正本の Markdown を、Vite プラグイン（`app/vite/legalMarkdown.ts`）が**ビルド時に HTML に変換**し、`virtual:legal/terms`・`virtual:legal/privacy` として読み込ませる。変換には `marked`（devDependency）を使う。
- 変換では、先頭の見出し（画面ではヘッダーに出す）と docs 向けの注記（引用）を除く。`terms.md`・`privacy.md` へのリンクは `/terms`・`/privacy` にし、それ以外の docs へのリンクは文字だけ残す。
- 本文はメインのバンドルに入れず、画面を開いたときに `import()` で読み込む（Suspense）。ヘッダーは先に出す。
- 見た目は既存の部品（AppHeader・MobileLayout）とデザイントークンで組む。

## 理由と検討した選択肢

| 選択肢 | 良い点 | 悪い点 |
| --- | --- | --- |
| ビルド時に変換（採用） | 正本が1か所のまま。実行時の依存が増えない | Vite プラグインを1つ持つ |
| 実行時に描画（react-markdown + remark-gfm） | React 要素として描ける | 依存とバンドルが数十KB増える |
| TSX に手で書き写す | 依存なし | 本文が2か所になり、改定のたびに手で同期が要る |

本文（約20KB）をメインのバンドルに入れると 500KB の警告を超えるため、本文は遅延読み込みにした。

## 影響

- [tech/frontend.md](../tech/frontend.md)（FE-1）の `/terms`・`/privacy` の行、[app/CLAUDE.md](../../app/CLAUDE.md) の技術スタック
- 実装：`app/vite/legalMarkdown.ts`、`app/src/features/legal/pages/LegalPage.tsx`、`app/src/routes.tsx`
- 規約を改定したら、`docs/legal/*.md` を直してビルドし直すだけで画面に反映される
