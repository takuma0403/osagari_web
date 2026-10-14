# ADR 0004: 部品カタログに Storybook を使い、下部ナビはデザインどおり4画面だけに出す

- 日付: 2026-10-14
- 状態: 採用（決定1の「CI での検査はしない」は [ADR 0005](0005-storybook-tests-in-ci.md) で変更）
- 関連する章ID: DS-1, DS-3, FE-1

## 背景

issue #7（DS-3 の共通部品）で、次の2点を決める必要があった。

1. issue の作業内容にある「部品一覧を確認できる簡易ページ（開発時のみ）」をどう作るか。
2. FE-1 のルート表では利用者向けの全画面が「スマホ＋下部ナビ」になっていたが、デザイン（[DS-4](../design/README.md)）で下部ナビがあるのは Main・Search・DealList・MyPage の4画面だけだった。

## 決定

1. 部品カタログは **Storybook**（`storybook`・`@storybook/react-vite`・`@storybook/addon-a11y`）で作る。自作の `/dev/components` ページは作らない。
   - 各部品に `<Name>.stories.tsx` を置き、状態ごとの見た目を並べる。既定のビューポートは 390px。
   - ローカルでの確認用とし、公開（Chromatic など）や CI での検査はしない。→ CI での検査は [ADR 0005](0005-storybook-tests-in-ci.md) で行うことにした。
2. 下部ナビ（`BottomNav`）は `/`・`/search`・`/deals`・`/mypage` だけに出す。商品詳細・出品・取引詳細・学生証登録・売上管理などには出さない。

## 理由と検討した選択肢

| 選択肢 | 良い点 | 悪い点 |
| --- | --- | --- |
| 自作の `/dev/components` ページ | 依存が増えない | 状態の切り替え・a11y の確認を自作することになる。本番ビルドから外す工夫が要る |
| **Storybook（採用）** | 状態ごとの表示・props の切り替え・a11y 検査（addon-a11y）がそろう。アプリのビルドとは別なので本番に混ざらない | devDependencies が増える（無料、費用はかからない） |

下部ナビは、デザインを正とした（FE-1 の表は「BottomNav ができてから表示する」としており、どの画面に出すかの判断はデザインに合わせる前提だった）。詳細や入力の画面は下部に固定の操作ボタンを置くため、ナビがあると重なる。

## 影響

- 更新したドキュメント：[tech/frontend.md](../tech/frontend.md) の FE-1（ルート表のレイアウト列）、[design/README.md](../design/README.md) の DS-3（Storybook・補助部品・Button と Badge の種類）、[app/CLAUDE.md](../../app/CLAUDE.md)（技術スタック・コマンド・構成）
- 実装への影響：`app/.storybook/`、`npm run storybook`・`npm run build-storybook`。`app/src/routes.tsx` で利用者向けのルートを下部ナビあり・なしの2グループに分けた
