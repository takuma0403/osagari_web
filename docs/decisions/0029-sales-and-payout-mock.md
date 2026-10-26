# ADR 0029: 売上の明細・出金申請のモックと、マイページの出品一覧

- 日付: 2026-10-27
- 状態: 採用
- 関連する章ID: PAY, PAY-2, PAY-5, JOBS-2, AUTH-3, DB

## 背景

マイページ・売上API（#32）を作るにあたり、次のことが決まっていなかった。

- 売上管理（[Sales](../design/screens/Sales.dc.html)）に出す「出金できる売上」「受け渡し待ち」「出金済み」を、どのデータから出すか。
- 出金申請（「出金を申請する」）をどこまで作るか。[ADR 0021](0021-transaction-payment-columns.md) は「出金は売上画面の出金申請（PAY、M5）で足す」としていたが、形は決めていなかった。
- マイページ（[MyPage](../design/screens/MyPage.dc.html)）の「出品した商品」を返す API がなかった（`GET /items` はほかの人向けの公開中の一覧）。

手数料率と端数は [ADR 0017](0017-fee-rounding-and-provisional-rate.md) の暫定（5%・切り捨て）のまま進めることを、ユーザーが確認した（料率そのものは未決のまま）。

## 決定

- **売上の明細**は、本人が出品者の取引のうち成立したもの（`in_deal`・`completed`）を1件1行で出す。キャンセルした取引は返金済みなので出さない。金額・料率・手数料・売上は取引成立時に記録した値をそのまま返し、計算し直さない（[ADR 0003](0003-fee-rate-applied-at-deal.md)）。
- 明細の状態（`SalesEntryState`）は次のとおり。集計はそれぞれの売上の合計。
  - `awaiting_handover` 受け渡し待ち：取引中
  - `settled` 確定：受け渡し完了・未出金（出金できる売上）
  - `paid_out` 出金済み：出金（`payout`）の記録がある
- **出金申請**（`POST /me/sales/payouts`）は、`settled` の取引をすべてまとめて出金し、取引ごとに `payout` の Payment を1行記録する（`payment.Gateway` に `Payout` を追加。モックは記録するだけ）。出金できる売上がなければ 409 `invalid_status_transition`。
- 二重の出金は、`payments (transaction_id) WHERE kind = 'payout'` の一意インデックスで防ぐ。同時に申請されたら、後の方は 409 になる。
- 出金の最低額・回数の上限・受け取り先（銀行口座）の登録は作らない。画面の受け取り先は「未登録」の固定表示にする（決済サービスの選定が[未決](../product/open-questions.md)のため）。
- **マイページの出品**は `GET /me/items` を新設し、本人のすべての出品（下書き・取り下げ・停止を含む）を最後に更新した順に、承認待ちの購入希望の件数つきで返す。**購入した商品**は既存の `GET /transactions?role=buyer`（[ADR 0028](0028-transaction-list.md)）を使う。プロフィール・有効期限は既存の `GET /me`。

## 理由と検討した選択肢

| 選択肢 | 良い点 | 悪い点 |
| --- | --- | --- |
| 状態を取引のステータスと payout の有無から出す（採用） | 新しい列が要らない。既存の記録だけで決まる | 明細のたびに payments を引く |
| transactions に出金日時の列を足す | 1テーブルで済む | 決済の記録が2か所に分かれる |
| 確定した売上をまとめて出金（採用） | 画面のボタン1つと合う。モックで十分 | 一部だけ出金できない |
| 金額を指定して出金 | 本番に近い | 取引ごとの記録（ADR 0021）と合わない。決済サービスが未定 |
| 一意インデックスで二重出金を防ぐ（採用） | 同時の申請でも確実 | マイグレーションが1つ増える |
| アプリ側の確認だけで防ぐ | スキーマが変わらない | 同時の申請で二重に記録しうる |
| `/me/items` を新設し購入は既存の一覧（採用） | 重複する API を作らない | app が2つの API を呼ぶ |
| `/me/purchases` も新設 | マイページの API がそろう | 取引一覧とほぼ同じ API が2つになる |

## 影響

- [product/06-payment.md](../product/06-payment.md)：PAY-5 を追加
- [tech/jobs-payment-mock.md](../tech/jobs-payment-mock.md) JOBS-2：操作の表に出金
- [tech/database.md](../tech/database.md) DB：二重の出金の防止
- [glossary.md](../glossary.md)：`SalesEntryState`、売上の明細
- [api/openapi.yaml](../api/openapi.yaml)：`GET /me/items`、`GET /me/sales`、`POST /me/sales/payouts`
- 実装：`000016_create_payments_payout_index`、`payment.Gateway.Payout`、`service.SalesService`、`ItemService.ListOwn`（#32）。画面は #33・#34
