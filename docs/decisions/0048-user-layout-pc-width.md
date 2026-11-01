# ADR 0048: 利用者向け画面の PC 幅レイアウト

- 日付: 2026-11-02
- 状態: 採用
- 関連する章ID: DS-1, DS-3, FE

## 背景

利用者向け画面は 390px のスマホを基準にデザインされており、PC で開くと最大幅 480px の列を中央に置くだけだった（`MobileLayout`）。1280px 幅では左右が大きく空く。Claude Design のキャンバスには PC 幅の利用者画面がないため、既存の部品で補う方針を決める必要があった（#128）。

## 決定

- ブレークポイントは 640px と 1024px。値は DS-1 に書き、CSS には数値を直接書く（CSS 変数は `@media` の条件に使えないため、トークンにはしない）。
- 列の最大幅は、商品一覧・検索結果が 1080px、それ以外が 560px。BottomSheet の幅も 560px にそろえる。
- 1024px 以上では `BottomNav` を上部の横ナビ（ロゴ＋アプリ名＋5項目）にする。出す画面はスマホと同じ4画面（ADR 0004 のまま）。
- 上部に置くときは、DOM 上でもナビを本文の前に移す。CSS の `order` で見た目だけ動かすと、Tab の順が「本文 → 上部ナビ」になり見た目と食い違うため（DS-1 のアクセシビリティ）。幅は `matchMedia` で見る（`lib/useMediaQuery.ts`）。
- 商品一覧のグリッドは 2・3・4列、検索結果は 1024px 以上で 2列。
- PC 幅も扱うので、レイアウトの名前を `MobileLayout` から `UserLayout` に変える。

## 理由と検討した選択肢

| 選択肢 | 良い点 | 悪い点 |
| --- | --- | --- |
| 上部の横ナビ（採用） | 一般的な Web サイトの形で、ナビの項目名を横に読める | 商品一覧のロゴ行と重なるため、PC では一覧側のロゴ行を隠す調整が要る |
| 左の縦ナビ | `BottomNav` を CSS だけで縦に並べられる | 一覧の横幅が狭くなる |
| ブレークポイント 768px・1024px | タブレットの縦向きと合う | 640〜767px で 2列のまま間延びする |
| ブレークポイント 640px・1024px（採用） | 小さめのタブレットから列を増やせる | — |
| DOM の位置を JS で切り替える（採用） | Tab の順と見た目が一致する | 幅の判定に JS が要る（matchMedia がない環境ではスマホの並び） |
| CSS の `order` だけで動かす | JS が要らない | PC で Tab の順が見た目と食い違う |

## 影響

- 更新したドキュメント：[design/README.md](../design/README.md)（DS-1、DS-3 の `BottomNav`）、[tech/frontend.md](../tech/frontend.md)（FE-1）、[app/CLAUDE.md](../../app/CLAUDE.md)。
- 実装：`app/src/layouts/UserLayout`（`MobileLayout` から改名、`nav`・`wide`）、`BottomNav` の `placement`、`lib/useMediaQuery.ts`、商品一覧・検索結果の CSS、`Modal` の BottomSheet の幅、Storybook のビューポート（#128）。
- ADR 0038 などにある `MobileLayout` の記述は、当時の記録としてそのまま残す。
