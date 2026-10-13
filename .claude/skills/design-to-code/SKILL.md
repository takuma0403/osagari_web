---
name: design-to-code
description: docs/design/screens の Claude Design（.dc.html）の画面を、React + CSS Modules + デザイントークン（CSS変数）の実装に移す手順。画面やUI部品を実装・修正するときに必ず使う。
argument-hint: "<画面名 例: DealQR>"
---

# design-to-code：デザインを実装に移す

## 参照するもの

- [docs/design/README.md](../../../docs/design/README.md)：DS-1 基本方針、DS-2 トークン、DS-3 共通部品、DS-4 画面と要件の対応
- `docs/design/screens/<画面>.dc.html`：その画面のマークアップと文言
- `app/CLAUDE.md`：ディレクトリ構成と規約

## `.dc.html` の読み方

- `<x-dc>` の中がマークアップ。`style="…"` の inline style が見た目のすべて。
- `{{name}}` は末尾の `<script type="text/x-dc">` の `renderVals()` が返す値。リストのサンプルデータや、状態ごとに切り替わるスタイル文字列もここにある。
- `<sc-for list="{{items}}" as="it">` は繰り返し、`<sc-if value="{{flag}}">` は条件表示。
- `data-props` は画面のバリエーション（例 `mode`：新規登録／再設定、`error`、`locked`）。**バリエーションごとの表示はすべて実装する対象。**
- `href="X.dc.html"` は仮の画面遷移。実際のルートは app の routes に従う。
- サンプルの文言（商品名、価格、学籍番号）は表示例。UIの固定文言（ボタン名、注意書き、説明文）は**一字一句そのまま使う**。

## 手順

1. DS-4 で画面に対応する章IDを確認し、要件と食い違いがないか確かめる（食い違ったら要件を優先し、ユーザーに報告する）。
2. 画面を部品に分ける。DS-3 の共通部品に当たるものは `app/src/components/` の既存部品を使う（なければ共通部品として作る）。
3. inline style を CSS Modules に移す：
   - 色・角丸・フォント・主要な余白は DS-2 の CSS 変数に置き換える（`#1F7A80` → `var(--color-primary)`）
   - 表にない値が出てきたら、近いトークンに寄せるか、DS-2 に追加してから使う（`spec-update` スキル）
   - 固定の `width: 390px; height: 844px` は使わず、幅100%・最大幅で流体にする。スクロール領域とボトム固定バーの構造は保つ
4. マークアップの意味を保つ：`<button>`、`<a>`、`<label for>`、`<dl>`、`<fieldset>/<legend>`、`role="alert" | "status" | "dialog"`、`aria-label` を残す。
5. 状態（読み込み中・空・エラー・無効）を、デザインにないものも含めて用意する。デザインにない状態は既存の InfoBox や Button の無効スタイルで表す。
6. Vitest + Testing Library で、主要な表示とバリエーションの切り替えをテストする。

## 確認

- `npm run typecheck` と `npm test` が通る。
- 開発サーバーで 390px 幅で表示し、デザインと見比べる（`run` スキルやブラウザツールが使える場合）。見比べられなかったときは、そのことを報告する。
- `grep -rnE '#[0-9A-Fa-f]{3,6}\b' app/src --include=*.module.css` で、トークン以外の色の直書きがないこと。
