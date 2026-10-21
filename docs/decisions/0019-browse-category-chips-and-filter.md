# ADR 0019: 一覧のカテゴリの分け方と検索の絞り込み

- 日付: 2026-10-22
- 状態: 採用
- 関連する章ID: BROWSE, DS-2, DS-3, DS-4, FE-1

## 背景

要件（[ITEM](../product/03-listing.md)）と出品画面のカテゴリは3つ（`ItemCategory`：教科書・参考書／家具・家電／ゲーム・その他）だが、一覧（[Main](../design/screens/Main.dc.html)）のデザインは、カテゴリチップを「すべて・教科書・参考書・家具・家電・ゲーム・その他」の5つに、写真のカテゴリ色を「教科書・家電・家具・ゲーム」の4つに分けていた（glossary の注記）。
また、検索結果（[Search](../design/screens/Search.dc.html)）の「絞り込み」ボタンを押したあとの画面がデザインになかった。どちらも一覧・検索画面（#22）の実装時に決める必要があった。

## 決定

カテゴリチップ・カテゴリ色・続きの読み込み方は 2026-10-22 にユーザーが選び、絞り込みシートはキャンバスに描いたデザインをユーザーが承認した。

- 一覧のカテゴリチップは **「すべて」＋要件の3区分**（すべて／教科書・参考書／家具・家電／ゲーム・その他）にする。選んだカテゴリは URL（`/?category=`）に持つ。
- 写真がないときの地の色は、`textbook` → 教科書色（`--color-cat-textbook`）、`furniture_appliance` → **家具色**（`--color-cat-furniture`）、`game_other` → ゲーム色（`--color-cat-game`）。家電色（`--color-cat-appliance`）は商品の写真には使わない。
- 検索の絞り込みは、ボトムシート（[SearchFilter](../design/screens/SearchFilter.dc.html)）で、カテゴリ・授業名・価格帯（下限・上限）・受け渡し希望キャンパス・状態を選ぶ。キーワードは検索欄で入れる。下限が上限より大きければ検索せずにエラーを出す。
- 検索条件はすべて URL のクエリ（`/search?keyword=…&category=…`）に持ち、戻る・再読み込みで同じ結果を出す。キーワード以外の条件は検索結果の上にチップで出し、×で外せる。
- 一覧の右上のお知らせ（ベル）は、通知の機能が要件にないため置かない。
- 一覧・検索の続きは、末尾の「もっと見る」で `nextCursor` を使って読み込む。

## 理由と検討した選択肢

| 選択肢 | 良い点 | 悪い点 |
| --- | --- | --- |
| チップを要件の3区分にそろえる（採用） | API の `category` にそのまま対応し、server・DB の変更が要らない。出品画面と分け方が一致する | デザインの5区分より粗い |
| デザインどおり5区分にする | 「ゲーム」「その他」を分けて探せる | カテゴリを増やすには要件・API・DB の変更が要る |
| 家具・家電を家電色にする | — | 教科書の色（淡い緑青）と近く、一覧で見分けにくい |
| 絞り込みシートのデザインを描かずに実装する | 早い | デザインが正本という決まりから外れる |

## 影響

- 更新したドキュメント：[glossary.md](../glossary.md)（カテゴリの注記）、[design/README.md](../design/README.md)（DS-2 のカテゴリ色、DS-4 の Main・SearchFilter）、[tech/frontend.md](../tech/frontend.md)（FE-1 の `/`・`/search` のクエリ）、[design/screens/SearchFilter.dc.html](../design/screens/SearchFilter.dc.html)（追加）
- 実装：`app/src/lib/itemOptions.ts`（カテゴリ等の選択肢と `categoryTint`）、`app/src/features/items/`（MainPage・SearchPage・SearchFilterSheet）
