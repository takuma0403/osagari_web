# app（フロントエンド）

> このディレクトリで作業するときの規約。全体の決まりはルートの [CLAUDE.md](../CLAUDE.md)、技術方針は [docs/tech/frontend.md](../docs/tech/frontend.md)、見た目は [docs/design/README.md](../docs/design/README.md)。

## 技術スタック

| 項目 | 採用 | 出典 |
| --- | --- | --- |
| ビルド | Vite 8 | FE |
| UI | React 19 + TypeScript 6（strict） | FE, FE-1 |
| ルーティング | React Router（`/items/:shareId` などすべてSPAのルート） | FE |
| スタイル | **CSS Modules + CSS変数**（Tailwind・CSS-in-JS・UIライブラリは使わない） | FE-1 |
| APIの型 | openapi-typescript で `docs/api/openapi.yaml` から `src/api/schema.gen.ts` を生成（手書き禁止） | API-1 |
| APIクライアント | `fetch` の薄いラッパー（`src/api/client.ts`）。`credentials: 'same-origin'`、`X-Device-Id` ヘッダーを付ける（ヘッダーは認証の実装時に追加） | AUTHN-3 |
| QR | 表示はQR生成ライブラリ、読み取りはカメラを使うライブラリ（選ぶときは軽さとメンテ状況を確認） | FE |
| テスト | Vitest + Testing Library（jsdom） | — |
| Lint / Format | oxlint / Prettier（Markdown は対象外） | — |
| パッケージ管理 | npm（Node 24 LTS。リポジトリ直下の `.nvmrc`） | ADR 0002 |

## ディレクトリ構成（予定。現在は main.tsx・App.tsx・api/ だけ）

```
app/
  index.html
  src/
    main.tsx, App.tsx, routes.tsx
    styles/
      tokens.css        … docs/design/README.md の DS-2 と同じ名前の CSS 変数だけを置く
      global.css        … リセット、フォント、body
    api/
      schema.gen.ts     … 生成物（編集しない）
      client.ts         … fetch ラッパー、エラー（Error スキーマ）の扱い
    components/         … DS-3 の共通部品（Button, TextField, CodeInput, ItemCard, BottomNav …）
      Button/Button.tsx, Button.module.css, Button.test.tsx
    features/<機能>/     … auth, items, deals, sell, student-card, mypage, sales, admin
      pages/            … ルートに対応する画面
      components/       … その機能だけで使う部品
      hooks/, api.ts
    lib/                … 汎用関数（金額表示、日付、共有文の生成など）
  public/
```

## 規約

- **色・余白・角丸・フォントは必ず `var(--…)` を使う。** 16進カラーや px の直書きは tokens.css 以外に書かない（レイアウト上の幅などは除く）。
- 画面は **390px 幅のスマホを基準**に作り、PC では中央寄せの最大幅（例 480px）にする。管理画面（`features/admin`）だけはPC幅のレイアウト。
- タップ領域 44px 以上、アイコンボタンに `aria-label`、入力に `<label>`、エラーは `role="alert"`（[DS-1](../docs/design/README.md)）。
- 共通部品は `components/` を探してから作る。同じ見た目の部品を画面ごとに作らない。
- 金額は `lib/format.ts` の関数で `¥1,500` 形式にする（Outfit フォント）。手数料などの計算はサーバーの値を表示し、フロントで計算しない（出品画面の見込み表示を除く）。
- 学籍番号の入力は8桁の数字だけを受け付け、ドメイン部分は固定表示（EmailField）。最終的な検証はサーバーが行う。
- 端末識別子は localStorage に保存する。Cookie は `__session` 以外を使わない（[AUTHN-3](../docs/tech/auth.md)）。
- 画像はアップロード前にブラウザで長辺1600px・JPEG/WebPに縮小し、APIが発行した署名付きURLへ直接 PUT する（[STORAGE](../docs/tech/storage.md)）。
- 共有ボタンは「【お下がり】商品名 ¥価格（状態） URL」の文をコピーし、スマホでは Web Share API も使う（[FE](../docs/tech/frontend.md)）。
- 画面の実装手順は `design-to-code` スキル、API を増やすときは `api-contract` スキル。

## コマンド

`app/` で実行する。最初に `nvm use`（未インストールなら `nvm install`）。

| 目的 | コマンド |
| --- | --- |
| 依存インストール | `npm ci` |
| 開発サーバー | `npm run dev`（http://localhost:5173。`/api` は Vite の proxy で `localhost:8080` へ。先に server の compose を起動する） |
| 型チェック | `npm run typecheck` |
| Lint / Format | `npm run lint` / `npm run format`（CI では `npm run format:check`） |
| テスト | `npm test` |
| APIの型生成 | `npm run gen:api`（openapi-typescript は TS 5 を peer に指定しているため、package.json の overrides で TS 6 を使わせている） |
| ビルド | `npm run build` |
