# ADR 0030: マイページの購入履歴の範囲と、出品の行の状態・行き先

- 日付: 2026-10-27
- 状態: 採用
- 関連する章ID: AUTH-3, ITEM, DEAL-1, DS-3

## 背景

マイページ画面（#33、[MyPage](../design/screens/MyPage.dc.html)）を作るにあたり、次のことが決まっていなかった。

- 「購入した商品」に、購入側のどの取引を出すか（[ADR 0029](0029-sales-and-payout-mock.md) は `GET /transactions?role=buyer` を使うとだけ決めていた）。
- 「出品した商品」の行に出す状態の文言と、行を押したときの行き先。デザインにあるのは「購入希望 n件」「取引中」「下書き」の3つだけ。
- デザインのタブは下線式で、DS-3 の `SegmentedTabs`（pill の塗り）と見た目が違う。

## 決定

- **購入した商品**は、購入側の取引のうち成立したもの（取引中 `in_deal`・完了 `completed`）だけを出す。承認待ち・却下・期限切れ・キャンセルは取引一覧（DealList）で見る。タブの件数も同じ範囲で数えるため、app は終了した取引（`phase=ended`）を `limit=50` で最後のページまで読み、絞り込む。行は取引詳細（`/deals/:dealId`）へ進み、バッジは取引一覧の購入側と同じ。
- **出品した商品**の行の状態と行き先は次のとおり。

  | 商品ステータス | バッジ | 行き先 |
  | --- | --- | --- |
  | `requested` | 購入希望 n件（warning） | 購入希望の一覧 `/sell/:itemId/requests` |
  | `draft` | 下書き（neutral） | 出品の編集 `/sell/:itemId` |
  | `withdrawn` | 取り下げ（neutral） | 出品の編集 |
  | `suspended` | 停止中（danger） | 出品の編集（編集できない案内を出す） |
  | `published` | 公開中（primary） | 商品ページ `/items/:shareId` |
  | `in_deal` | 取引中（primary） | 商品ページ |
  | `completed` | 売却済み（neutral） | 商品ページ |

  下書きで価格を入れていなければ「価格未設定」と出す。`GET /me/items` の行にはカテゴリがないため、写真がないときの地の色は教科書の色（`--color-cat-textbook`）にそろえる（デザインも同じ色）。
- タブは `SegmentedTabs` に下線版（`variant="underline"`）を足して、デザインどおりにする。プロフィールのアイコンは `Avatar` に 48px（`size="lg"`）と白地（`inverse`）を足して使う。

## 理由と検討した選択肢

| 選択肢 | 良い点 | 悪い点 |
| --- | --- | --- |
| 購入は取引中・完了だけ（採用） | 「購入した」の意味どおり。取引一覧と役割が分かれる | 件数のために終了した取引を全件読む |
| 購入側の取引をすべて | 取引一覧と同じ API の使い方で済む | 取引一覧とほぼ同じ内容になる。ページ送りのため件数が定まらない |
| 取引中・完了・キャンセル | 一度成立した購入が分かる | キャンセルは購入していないのに出る |
| SegmentedTabs に下線版を足す（採用） | キー操作などを共有し、デザインどおり | 部品の種類が1つ増える |
| 既存の SegmentedTabs をそのまま使う | 部品が増えない | デザインと見た目が違う |

## 影響

- [design/README.md](../design/README.md)：DS-3 の `SegmentedTabs`・`Avatar`、DS-4 のマイページの行
- API の変更はない（`limit` は既存のパラメーター）
- 実装：`app/src/features/mypage`、`SegmentedTabs` の `variant`、`Avatar` の `size`・`inverse`、`listTransactions` の `limit`（#33）
