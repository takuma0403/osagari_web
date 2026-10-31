# ADR 0039: 試験運用の初回ダイアログ

- 日付: 2026-11-01
- 状態: 採用
- 関連する章ID: OVERVIEW, PAY, FE-1, SCREEN, DS-3

## 背景

フェーズ1は試験運用で、決済はモック（[ADR 0036](0036-terms-and-privacy.md)、[利用規約](../legal/terms.md) 第２条）。規約に書くだけでは読まれないことがあるため、公開（M7）の前に、初めて開いた利用者へ必ず伝え、実際の売買に使われるのを防ぐ（#107）。Claude Design のキャンバスにはこのダイアログがない。

## 決定

- その端末で初めて開いたとき、ログイン前の画面・管理画面を含むすべての画面の上にダイアログを出す。`/terms`・`/privacy` を開いているときは重ねない。
- 文言は利用規約 第２条の要旨だけにする（見出し「試験運用中です」、実際のお金の支払い・受け取りは発生しない／実際の売買に使わない、ボタン「確認した」）。規約へのリンクは置かない。
- 閉じるのは「確認した」ボタンだけで、Esc・背面の幕では閉じない。
- 確認したことは localStorage（キー `osagari.trialNoticeAcknowledged`）に保存する。Cookie は `__session` 以外を使わない（AUTHN-3）。localStorage が使えなければ、そのページの間だけ閉じたままにする。
- 部品は新しく作らず、`ConfirmDialog` の取り消し（`cancelLabel`・`onCancel`）を省略できるようにして使う。省いたときは確定ボタンだけになり、Esc・幕では閉じない。デザインは後でキャンバスに足す。

## 理由と検討した選択肢

| 選択肢 | 良い点 | 悪い点 |
| --- | --- | --- |
| 「確認した」ボタンだけで閉じる（採用） | 読まずに閉じて既読になるのを防げる | ほかのダイアログと閉じ方が違う |
| Esc・幕でも閉じて既読にする | ほかのダイアログと同じ操作 | 読まずに既読になり得る |
| Esc・幕で閉じるが既読にしない | 一時的に閉じられる | 読まずに使い続けられ、毎回出て煩わしい |
| 規約へのリンクを置く | 全文をすぐ読める | 要旨で足り、ダイアログを出したまま移動する扱いが増える |
| 確認済みを Cookie に保存 | サーバーでも分かる | `__session` 以外の Cookie は使わない決まり（Firebase Hosting が落とす） |

## 影響

- 更新：[tech/frontend.md](../tech/frontend.md) FE-1、[product/08-screens.md](../product/08-screens.md) SCREEN、[design/README.md](../design/README.md) DS-3。
- 実装：`app/src/components/ConfirmDialog`（取り消しを省略可能に）、`app/src/lib/trialNotice.ts`、`app/src/features/legal/components/TrialNotice.tsx`、`app/src/routes.tsx`（全画面の親ルート `RootLayout`）。
- 本番の決済を始めて試験運用を終えるときは、このダイアログを外す。
