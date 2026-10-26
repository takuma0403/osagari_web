# ADR 0028: 取引一覧の行の単位と取得の分け方

- 日付: 2026-10-26
- 状態: 採用
- 関連する章ID: DEAL-1, DEAL-3, DEAL-4, API-1

## 背景

取引一覧（DealList）の API（#47）を作るにあたり、デザインと仕様から次の点が読み取れなかった。

- 出品タブの行の単位。デザインでは出品側の購入希望が「購入希望 3件／相手を選んでください」という商品ごとの1行で、取引（Transaction）ごとではない。
- 出品側で、選ばれなかった（却下・期限切れの）購入希望を一覧に出すか。デザインの出品タブの終了欄は「完了」「キャンセル」だけ。
- 購入／出品、進行中／終了を1回で取るか分けて取るか、件数が増えたときのページ分け。

## 決定

- `GET /transactions?role=buyer|seller&phase=active|ended` で、ログイン中の本人が当事者の取引だけを返す。学籍番号・手数料は含めない（DEAL-1）。
- 区分ごとに出すもの：

  | role / phase | 出すもの |
  | --- | --- |
  | buyer / active | 購入希望（requested）・取引中（in_deal）。取引ごと |
  | buyer / ended | 却下・期限切れ・キャンセル・完了 |
  | seller / active | 承認待ちの購入希望を**商品ごとに1行**（件数つき、商品ID で購入希望の一覧 `/sell/:itemId/requests` へ）と、取引中 |
  | seller / ended | キャンセル・完了（出品側の却下・期限切れは出さない） |

- 並びは最後に動いた日時（`updated_at`。商品ごとの行は最新の購入希望を出した日時）の新しい順。
- 承認期限を過ぎた購入希望も、短周期ジョブ（JOBS-1）で期限切れになるまでは承認待ちとして出す（商品ページの `myRequest` と同じ。期限を過ぎたものは承認できない。ADR 0022）。
- 進行中（active）は全件を返す。購入希望は48時間で期限切れになり、取引中は商品ごとに1件なので件数が限られるため。終了（ended）は商品の一覧と同じ cursor 方式（`updated_at` と ID）でページ分けする（既定20件、最大50件）。

## 理由と検討した選択肢

| 選択肢 | 良い点 | 悪い点 |
| --- | --- | --- |
| 出品側の購入希望は商品ごと（採用） | デザインどおり。出品者は商品単位で相手を選ぶ（DEAL-1）ので、行の先が購入希望の一覧と一致する | 行の形が取引ごとの行と少し違う（取引ID が null、件数あり） |
| すべて取引ごと | 行の形がそろう | デザインと違い、同じ商品が件数分並ぶ |
| role＋phase で分けて取得・終了だけ cursor（採用） | 終了した取引が増えても重くならない。画面のセクションと対応する | 1画面で2回呼ぶ |
| role だけで全件 | 1回で済む | 終了した取引が増え続けると応答が大きくなる |

## 影響

- [api/openapi.yaml](../api/openapi.yaml)：`listTransactions`、`TransactionList`・`TransactionListEntry`・`TransactionListItem`・`TransactionPhase`
- [glossary.md](../glossary.md)：取引一覧の区分（`TransactionPhase`）
- [product/08-screens.md](../product/08-screens.md)：取引一覧の行からこの ADR へリンク
- 実装：`server/internal/service/deal_list.go`、`TransactionRepository.ListByParty`・`ListRequestGroupsBySeller`、`ItemRepository.FindByIDs`。画面は #48
