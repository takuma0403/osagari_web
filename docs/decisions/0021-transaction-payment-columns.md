# ADR 0021: 取引・決済のテーブルの支払い方法・決済ステータス・出金の持ち方

- 日付: 2026-10-22
- 状態: 採用
- 関連する章ID: DATA, PAY, JOBS-2, DB

## 背景

取引と決済モックのテーブル（#24）を作るにあたり、[DATA](../product/09-data-model.md) と [用語集](../glossary.md) だけでは次のことが決まっていなかった。

- Payment の「ステータス」の値（`PaymentStatus` の定義がない）。
- 購入希望のときに購入者が選ぶ支払い方法（[Request](../design/screens/Request.dc.html) のクレジットカード・PayPay）を、どこに持つか。DATA では Payment にだけある。
- 出金（`payout`）の Payment が取引に結び付くのか、ユーザー単位なのか。

## 決定

- `PaymentStatus` は `pending` 処理中 / `succeeded` 成功 / `failed` 失敗の3値にする。モックは常に `succeeded` で記録する。
- 支払い方法は `transactions.payment_method`（必須）と `payments.payment_method` の両方に持つ。`payments` では売上の確定（`settle`）と出金（`payout`）のときは空にする。
- 出金も取引ごとに1行の Payment として記録する（売上画面の「出金済み」が取引ごとに表示されるため）。`payments.transaction_id` は常に必須にする。
- 決済の窓口 `payment.Gateway` は枠の確保・預かり・枠の解放・返金・売上の確定の5つにする。出金は売上画面の出金申請（PAY、M5）で足す。
- Webhook の受け口は `POST /api/webhooks/payment` として枠だけ作り、フェーズ1は何もせず 204 を返す。

## 理由と検討した選択肢

| 選択肢 | 良い点 | 悪い点 |
| --- | --- | --- |
| ステータス3値（採用） | 非同期で確定する本番の決済（PayPay など）にもそのまま使える | フェーズ1では `pending` を使わない |
| ステータス2値（`succeeded`/`failed`） | 最小 | 本番に切り替えるときにスキーマ変更が要る |
| 支払い方法を両方に持つ（採用） | Gateway の各操作に取引を渡すだけで済む。取引一覧で支払い方法を出せる | 同じ値を2か所に持つ |
| 支払い方法を Payment だけに持つ | DATA のまま | 預かり以降の操作のたびに枠の確保の行を引く必要がある |
| 出金を取引ごとの行にする（採用） | `transaction_id` を必須にでき、売上一覧と突き合わせやすい | 複数の取引をまとめて出金する場合は複数行になる |
| 出金をユーザー単位の行にする | まとめ出金を1行で表せる | `user_id` 列と `transaction_id` を空にする条件が増える |

## 影響

- [glossary.md](../glossary.md)：`PaymentStatus` を追加、Payment の説明
- [product/09-data-model.md](../product/09-data-model.md) DATA：Transaction に支払い方法、Payment の支払い方法とステータス
- [tech/jobs-payment-mock.md](../tech/jobs-payment-mock.md) JOBS-2：Webhook の枠のパス、モックの記録内容
- 実装：`transactions`・`payments` のマイグレーション、`domain.PaymentStatus`、`internal/payment`（Gateway と mock）、`POST /webhooks/payment`（#24）
