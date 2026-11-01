# ADR 0005: Storybook の全 story を PR の CI で描画・a11y 検査する

- 日付: 2026-10-14
- 状態: 採用
- 関連する章ID: DS-1, DS-3

## 背景

[ADR 0004](0004-storybook-and-bottom-nav.md) では、Storybook をローカルの確認用とし「CI での検査はしない」とした。addon-a11y の `parameters.a11y.test: 'error'` は設定済みだが、Storybook を開かないと違反に気づけない。

M1 以降は画面の実装とともに部品と story が増える。表示の崩れや DS-1 のアクセシビリティ違反を、増える前の基盤の段階から PR で検出できるようにしたい（issue #50）。

## 決定

1. `@storybook/addon-vitest` で全 story を Vitest のテストとして実行する。ブラウザモード（`@vitest/browser-playwright` の Chromium、headless）で描画し、addon-a11y の検査で違反があればテストを失敗にする。
2. Vitest は `app/vite.config.ts` の `test.projects` で2つに分ける。
   - `unit`：既存の jsdom の単体テスト（`npm test`）
   - `storybook`：story のテスト（`npm run test:storybook`）
3. `.github/workflows/ci.yml` の app job で、Chromium をインストール（Playwright のバージョンごとにキャッシュ）してから `npm run test:storybook` と `npm run build-storybook` を実行する。
4. Storybook の公開（Chromatic などの見た目の差分検査）はしない（ADR 0004 のまま）。

## 理由と検討した選択肢

| 選択肢 | 良い点 | 悪い点 |
| --- | --- | --- |
| CI で検査しない（ADR 0004 のまま） | CI の時間が増えない | a11y 違反や描画エラーが PR で見つからない |
| `@storybook/test-runner` | 実績が長い | Storybook をビルドして配信するサーバーが別に要る。Storybook 10 では addon-vitest が推奨 |
| **addon-vitest ＋ Vitest のブラウザモード（採用）** | 既存の Vitest にプロジェクトとして足せる。サーバー不要。preview.tsx の設定（ルーター・トークン・a11y）がそのまま効く | Chromium のインストールが要る |

費用（COST）：PR #51 の CI で実測すると、app job は追加前の約1分から、Chromium のキャッシュなしで1分57秒、キャッシュありで1分35秒になった（story のテスト約16〜19秒・19ファイル52 story、`build-storybook` 約4秒、Chromium のインストール約20秒）。増加は1回あたり0.5〜1分で、GitHub Actions の枠（Pro で月3,000分）に対して十分小さい。

## 影響

- 更新したドキュメント：[ADR 0004](0004-storybook-and-bottom-nav.md)（CI の部分をこの ADR で変更）、[design/README.md](../design/README.md) の DS-3、[app/CLAUDE.md](../../app/CLAUDE.md)（技術スタック・コマンド）
- 実装への影響：`app/vite.config.ts`（`test.projects`）、`app/.storybook/main.ts`（addon-vitest）、`app/package.json`（`test`・`test:watch` を `unit` に限定、`test:storybook` を追加）、`.github/workflows/ci.yml`
