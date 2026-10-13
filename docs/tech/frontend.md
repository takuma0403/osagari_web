# フロントエンド（技術）

> 関連: [一覧・検索・共有（要件）](../product/04-browse-search-share.md) / [デザイン](../design/README.md) / 実装規約: [app/CLAUDE.md](../../app/CLAUDE.md)

## FE フロントエンド

Vite + React のSPAを Firebase Hosting に置きます。ログイン必須のアプリなので、検索エンジン向けのSSRは必要ありません。Firebase Hosting は無料のSparkプランで使えます。

| 選択肢 | 費用 | 判断 |
| --- | --- | --- |
| Vite + React（SPA）を Firebase Hosting に置く | 無料（Sparkプランで可） | **採用** |
| Next.js を Firebase App Hosting に置く | Blazeプラン（従量課金）が必須。無料枠あり | SSRが必要になったら検討 |
| Next.js を Vercel の Hobby プランに置く | 無料だが非商用に限る | **不採用**。手数料を取るので商用扱いになり、有料のProプランが必要 |

- Firebase Hosting 単体では Next.js のSSRは動きません。Next.js をSSRで使うには、Blazeプランが前提の App Hosting を使う必要があります（[Firebase公式ブログ](https://firebase.blog/posts/2024/05/app-hosting-vs-hosting)）。
- Vercel の Hobby プランは非商用の個人利用に限られ、商用利用には Pro 以上が必要です（[Vercel Fair Use Guidelines](https://vercel.com/docs/limits/fair-use-guidelines)）。
- **ルーティング**：画面遷移は React Router で行います。商品ページの共有リンク（`/items/:id`）もSPAのルートとして扱い、Firebase Hosting では `/api/**` 以外のすべてのパスを `index.html` に返す設定にします。
- **共有リンクのプレビュー**：SPAのため、TeamsやLINEにリンクを貼っても商品ごとのタイトルや画像のプレビュー（OGP）は表示されず、アプリ共通の表示になります。リンクを開けば商品ページは問題なく表示されます。その代わりに、共有ボタンでは商品名・価格・状態とURLをまとめた文章をコピーできるようにし、リンクを貼るだけで内容が伝わるようにします（例：「【お下がり】線形代数学の教科書 ¥1,500（書き込みなし） https://…/items/123」）。スマホでは Web Share API で共有シートも開けるようにします。
- **QRコード**：表示はQR生成ライブラリ、読み取りはブラウザのカメラを使うライブラリで行います。カメラの利用にはHTTPSが必要ですが、Firebase Hosting は標準でHTTPSです。

### FE-1 実装方針（ハーネス整備時の決定）

元文書に追加した決定事項です（[ADR 0001](../decisions/0001-docs-structure.md)）。

- 言語は TypeScript。スタイリングは **CSS Modules + CSS変数**（デザイントークンは [design/README.md](../design/README.md) を正とし、`app/src/styles/tokens.css` に1対1で写す）。
- APIの型は [docs/api/openapi.yaml](../api/openapi.yaml) から openapi-typescript で生成し、手書きしない。
