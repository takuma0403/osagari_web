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
- **ルーティング**：画面遷移は React Router で行います。商品ページの共有リンク（`/items/:shareId`）もSPAのルートとして扱い、Firebase Hosting では `/api/**` 以外のすべてのパスを `index.html` に返す設定にします。
- **共有リンクのプレビュー**：SPAのため、TeamsやLINEにリンクを貼っても商品ごとのタイトルや画像のプレビュー（OGP）は表示されず、アプリ共通の表示になります。リンクを開けば商品ページは問題なく表示されます。その代わりに、共有ボタンでは商品名・価格・状態とURLをまとめた文章をコピーできるようにし、リンクを貼るだけで内容が伝わるようにします（例：「【お下がり】線形代数学の教科書 ¥1,500（書き込みなし） https://…/items/123」）。スマホでは Web Share API で共有シートも開けるようにします。
- **QRコード**：表示はQR生成ライブラリ、読み取りはブラウザのカメラを使うライブラリで行います。カメラの利用にはHTTPSが必要ですが、Firebase Hosting は標準でHTTPSです。

### FE-1 実装方針（ハーネス整備時の決定）

元文書に追加した決定事項です（[ADR 0001](../decisions/0001-docs-structure.md)）。

- 言語は TypeScript。スタイリングは **CSS Modules + CSS変数**（デザイントークンは [design/README.md](../design/README.md) を正とし、`app/src/styles/tokens.css` に1対1で写す）。
- APIの型は [docs/api/openapi.yaml](../api/openapi.yaml) から openapi-typescript で生成し、手書きしない。
- 画面のルートは次のとおり（実装は `app/src/routes.tsx`。変えるときは両方を直す）。利用者向けは 390px 基準・PC では中央寄せの列（`MobileLayout`）、管理画面は PC 幅（`AdminLayout`）で表示する（[DS-1](../design/README.md)）。下部ナビ（DS-3 の `BottomNav`）は、デザインでナビのある商品一覧・検索結果・取引一覧・マイページ（Main・Search・DealList・MyPage）だけに出す（[ADR 0004](../decisions/0004-storybook-and-bottom-nav.md)）。

  | パス | 画面 | レイアウト |
  | --- | --- | --- |
  | `/login` | ログイン | スマホ |
  | `/signup`、`/signup/code`、`/signup/password` | 新規登録（メール入力 → 認証コード → パスワード設定） | スマホ |
  | `/password-reset`、`/password-reset/code`、`/password-reset/password` | パスワード再設定（同上） | スマホ |
  | `/recheck` | ログイン時の本人確認（AUTHN-3） | スマホ |
  | `/terms`、`/privacy` | 利用規約、プライバシーポリシー（ログイン画面からリンク。内容は [未決](../product/open-questions.md) で、決まるまでは仮ページ） | スマホ |
  | `/` | 商品一覧 | スマホ＋下部ナビ |
  | `/search` | 検索結果 | スマホ＋下部ナビ |
  | `/items/:shareId` | 商品詳細（共有リンク） | スマホ |
  | `/items/:shareId/request` | 購入希望・決済（モック） | スマホ |
  | `/sell`、`/sell/:itemId` | 出品、出品の編集（編集APIが商品IDを受け取るため、共有用IDではなく商品IDを使う。[ADR 0017](../decisions/0017-fee-rounding-and-provisional-rate.md)） | スマホ |
  | `/deals` | 取引一覧（下部ナビの「取引」） | スマホ＋下部ナビ |
  | `/deals/:dealId` | 取引詳細（購入希望の承認、受け渡しQR、キャンセル） | スマホ |
  | `/deals/:dealId/scan` | QR読み取り（購入者） | スマホ |
  | `/deals/:dealId/complete` | 完了リンク・完了コードの入力（購入者、DEAL-3）。トークンの載せ方は DEAL-3 の実装で決める | スマホ |
  | `/mypage` | マイページ | スマホ＋下部ナビ |
  | `/mypage/student-card`、`/mypage/sales` | 学生証登録、売上管理 | スマホ |
  | `/admin`（→ `/admin/student-cards`）、`/admin/users`、`/admin/items`、`/admin/settings` | 管理：学生証審査、ユーザー、出品・取引、設定 | PC |

- **ログインのガード**（[ADR 0013](../decisions/0013-frontend-auth-guard.md)）：起動時に `GET /me` でログイン状態を確かめ、どの API でも `401 unauthorized` が返ったら未ログインに戻す。
  - ログイン前だけの画面（`/login`、`/signup*`、`/password-reset*`、`/recheck`）は、ログイン済みなら商品一覧へ送る。`/terms`・`/privacy` は誰でも見られる。それ以外の画面はすべてログインを求め、未ログインならログイン画面へ送る。
  - ログイン画面へ送るときは開こうとしていたページ（パス）を location.state に残し、ログイン（再確認を含む）に成功したらそこへ戻す。なければ商品一覧。
  - 共有リンク（`/items/:shareId`）も、[未決事項](../product/open-questions.md)が決まるまではログインを求める。
  - 管理画面（`/admin/**`）は `GET /api/admin/session` で管理者かどうかを確かめ、404 なら「ページが見つからない」を出す。管理画面への導線は UI に出さない（[ADR 0015](../decisions/0015-admin-authorization.md)）。
