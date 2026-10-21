# ADR 0020: 共有リンクを未ログインで開いたときの扱いと、共有はコピーだけにすること

- 日付: 2026-10-22
- 状態: 採用
- 関連する章ID: BROWSE, FE, FE-1, DS-4

## 背景

未決事項「共有リンクをログインしていない人が開いたときの表示」（[product/open-questions.md](../product/open-questions.md)）は、[ADR 0013](0013-frontend-auth-guard.md) で「決まるまではログインを求める」としていた。
また、[FE](../tech/frontend.md) には「スマホでは Web Share API で共有シートも開ける」とあったが、デザイン（[Detail](../design/screens/Detail.dc.html)・[DetailShare](../design/screens/DetailShare.dc.html)）では「ほかのアプリで共有」が Tweak `shareApp`（既定オフ）に移っていた。どちらも商品詳細・共有画面（#23）の実装時に決める必要があった。

## 決定

2026-10-22 にユーザーが選んだ。

- 共有リンク（`/items/:shareId`）を未ログインで開いたときは、ほかの画面と同じく**ログインを求め**、ログイン（再確認を含む）に成功したら商品ページへ戻す。概要だけを見せる公開の表示は作らない。
- 共有は**文のコピーだけ**にする。Web Share API（「ほかのアプリで共有」ボタン・詳細右上の共有アイコン）は使わず、デザインの Tweak `shareApp` は実装しない。

## 理由と検討した選択肢

| 選択肢 | 良い点 | 悪い点 |
| --- | --- | --- |
| 未ログインはログインを求める（採用） | 学内限定の前提を崩さない。公開用の API が要らず、app だけで完結する | 学外の人やログインしていない人には中身が見えない |
| 未ログインでも概要だけ見せる | リンクを開いた人に内容が伝わる | 公開の API と表示を足す必要がある。学外に出品情報が出る |
| Web Share API も使う | スマホで送り先のアプリを選べる | 対応端末だけの表示になり、デザインの既定（オフ）とずれる。コピーで Teams などに貼れば足りる |
| コピーだけにする（採用） | どの端末でも同じ操作になる | 送り先のアプリは自分で開く必要がある |

## 影響

- 更新したドキュメント：[product/open-questions.md](../product/open-questions.md)、[product/04-browse-search-share.md](../product/04-browse-search-share.md)（BROWSE）、[tech/frontend.md](../tech/frontend.md)（FE・FE-1）、[design/README.md](../design/README.md)（DS-4 の Detail・DetailShare）、[README.md](../README.md)（対応表）、[app/CLAUDE.md](../../app/CLAUDE.md)
- 実装：`app/src/features/items/`（DetailPage・ShareSheet）、`app/src/routes.tsx`（`/items/:shareId` は RequireAuth の下のまま）
