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
| APIクライアント | `fetch` の薄いラッパー（`src/api/client.ts`）。`credentials: 'same-origin'`、`X-Device-Id` ヘッダー（`src/lib/deviceId.ts`）を付ける。失敗は `ApiError`（`status`・`code`・`message`）で投げ、通信自体の失敗はフロント専用の `network_error`（status 0） | AUTHN-3 |
| QR・バーコード | QR の表示は `uqr`（行列を SVG で描く）。読み取りは QR・ISBN のバーコードとも `@zxing/browser` を動的 import で使う（`src/lib/cameraScan.ts`） | FE, ADR 0025 |
| テスト | Vitest + Testing Library（jsdom）+ user-event。story は addon-vitest で Chromium（Playwright）に描画し a11y も検査 | DS-1, ADR 0005 |
| 部品カタログ | Storybook（`@storybook/react-vite` + addon-a11y + addon-vitest）。公開しない。全 story を CI で検査する | DS-3, ADR 0004, ADR 0005 |
| Lint / Format | oxlint / Prettier（Markdown は対象外） | — |
| パッケージ管理 | npm（Node 24 LTS。リポジトリ直下の `.nvmrc`） | ADR 0002 |

## ディレクトリ構成（features/ と DS-3 の部品は画面の実装とともに増える）

```
app/
  index.html
  src/
    main.tsx, App.tsx
    routes.tsx          … 全画面のルート（一覧は docs/tech/frontend.md の FE-1）
    styles/
      tokens.css        … docs/design/README.md の DS-2 と同じ名前の CSS 変数だけを置く
      global.css        … リセット、フォント、body
    api/
      schema.gen.ts     … 生成物（編集しない）
      client.ts         … fetch ラッパー、エラー（Error スキーマ）の扱い
    assets/             … 画面で使う画像（logo.png は docs/design/assets/logo.png を 128px に縮小したもの）
    layouts/            … MobileLayout（390px 基準・PC は中央寄せ）、AdminLayout（PC 幅）
    components/         … DS-3 の共通部品（Button, TextField, CodeInput, ItemCard, BottomNav …）
      Button/Button.tsx, Button.module.css, Button.test.tsx, Button.stories.tsx
    features/<機能>/     … auth, items, deals, sell, student-card, mypage, sales, admin
      pages/            … ルートに対応する画面
      components/       … その機能だけで使う部品
      hooks/, api.ts
    lib/                … 汎用関数（金額表示、日付、共有文の生成など）
  public/
  .storybook/           … Storybook の設定（トークン・フォントの読み込み、390px のビューポート、ルーター）
```

## 規約

- **色・余白・角丸・フォント・文字サイズは必ず `var(--…)` を使う（文字サイズは DS-2 の `--text-*`）。** 16進カラーや px の直書きは tokens.css 以外に書かない（レイアウト上の幅などは除く）。
- 画面は **390px 幅のスマホを基準**に作り、PC では中央寄せの最大幅（例 480px）にする。管理画面（`features/admin`）だけはPC幅のレイアウト。
- タップ領域 44px 以上、アイコンボタンに `aria-label`、入力に `<label>`、エラーは `role="alert"`（[DS-1](../docs/design/README.md)）。
- 共通部品は `components/` を探してから作る。同じ見た目の部品を画面ごとに作らない。部品を作ったら、状態ごと（無効・エラー・選択中など）の story を `<Name>.stories.tsx` に置く。
- 金額は `lib/format.ts` の関数で `¥1,500` 形式にする（Outfit フォント）。手数料などの計算はサーバーの値を表示し、フロントで計算しない（出品画面の見込み表示を除く）。
- 学籍番号の入力は8桁の数字だけを受け付け、ドメイン部分は固定表示（EmailField）。最終的な検証はサーバーが行う。
- 端末識別子は localStorage に保存する。Cookie は `__session` 以外を使わない（[AUTHN-3](../docs/tech/auth.md)）。
- 画像はアップロード前にブラウザで長辺1600px・JPEG/WebPに縮小し、APIが発行した署名付きURLへ直接 PUT する（[STORAGE](../docs/tech/storage.md)）。
- 共有ボタンは「【お下がり】商品名 ¥価格（状態） URL」の文をコピーする。Web Share API は使わない（[FE](../docs/tech/frontend.md)、ADR 0020）。
- 画面の実装手順は `design-to-code` スキル、API を増やすときは `api-contract` スキル。

## コマンド

`app/` で実行する。最初に `nvm use`（未インストールなら `nvm install`）。

| 目的 | コマンド |
| --- | --- |
| 依存インストール | `npm ci` |
| 開発サーバー | `npm run dev`（http://localhost:5173。`/api` は Vite の proxy で `localhost:8080` へ。先に server の compose を起動する） |
| 型チェック | `npm run typecheck` |
| Lint / Format | `npm run lint` / `npm run format`（CI では `npm run format:check`） |
| テスト | `npm test`（jsdom の単体テスト）/ `npm run test:storybook`（全 story の描画と a11y。初回は `npx playwright install chromium`） |
| APIの型生成 | `npm run gen:api`（openapi-typescript は TS 5 を peer に指定しているため、package.json の overrides で TS 6 を使わせている） |
| ビルド | `npm run build` |
| 部品カタログ | `npm run storybook`（http://localhost:6006）/ `npm run build-storybook` |
