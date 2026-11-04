# デザイン（UIの正本）

- **キャンバス（正本）**: https://claude.ai/artifact/QdafhFt85mz3WvYPPtDGEH （Claude Design「佐賀大お下がりフリマ デザイン」）
- **リポジトリ内の写し**: [screens/](screens/) … キャンバスの各アートボードの `.dc.html` ソースと [canvas.json](screens/canvas.json)（2026-10-22 取得、version `1792595429-a00f`）
- **ロゴ**: [assets/logo.png](assets/logo.png)（1254×1254。画面ではアイコン枠に `object-fit: cover; transform: scale(1.32)` で表示している。アプリでは 128×128 に縮小した `app/src/assets/logo.png` を `AppLogo` 部品で使う）
- 更新手順は [design-sync スキル](../../.claude/skills/design-sync/SKILL.md)、実装への移し方は [design-to-code スキル](../../.claude/skills/design-to-code/SKILL.md) を参照。

> `.dc.html` は Claude Design 独自のコンポーネント形式です（`<x-dc>` 内がマークアップ、`{{…}}` は末尾の `<script type="text/x-dc">` の `renderVals()` の値、`<sc-for>` は繰り返し、`<sc-if>` は条件表示、`data-props` は表示を切り替えるパラメータ）。**見た目と文言の参照元**として読み、コードとしてはそのまま流用しないこと。

## DS-1 基本方針

- **スマホ優先**：利用者向け画面は幅 390px（高さ 844px）のスマホを基準に作られている。管理画面だけはPC幅（1280px、流体レイアウト）。
- **利用者向け画面の PC 幅**：キャンバスには PC 幅の利用者画面がないため、既存の部品を組み替えて補う（[ADR 0048](../decisions/0048-user-layout-pc-width.md)）。390px の見た目は変えない。
  - ブレークポイントは **640px**（タブレット）と **1024px**（PC）。CSS 変数は `@media` の条件に使えないので、CSS には数値を直接書く。
  - 列の最大幅は **1080px** と **560px** の2つ。最大幅を超えたら、列の左右に `--color-border` の線を引く。BottomSheet の幅は 560px に合わせる。
    - 1080px：商品一覧・検索結果・取引一覧・マイページ・商品詳細・出品・取引の画面（購入希望、購入希望の一覧、取引詳細、QR読み取り、受け渡しの完了）。
    - 560px：学生証・売上管理・ログイン前の画面・規約。
  - 1024px 以上では、`BottomNav` を画面上部の横ナビにする（DS-3）。出す画面はスマホと同じ。商品一覧のヘッダーのロゴ行は、上部ナビと重なるので隠す。
  - 一覧の列数：
    - 商品一覧のグリッド（ItemCard）は、2列 → 640px 以上で 3列 → 1024px 以上で 4列。
    - 検索結果（ItemRow）・取引一覧の行・マイページの行と設定のリンクは、640px 以上で 2列。取引一覧の購入・出品のタブは最大 400px。
  - 1024px 以上の2カラム（`TwoColumn`）：
    - 商品詳細：左に写真（360px の正方形・角丸。本文をスクロールしても上に残す）、右に商品の情報と出品者。下部のボタンは右カラムにそろえる（[ADR 0049](../decisions/0049-item-detail-pc-two-column.md)）。
    - 出品：左に写真欄（360px）、右に入力欄。下部のボタンは右カラムにそろえる。
    - 取引の画面：左に商品のカード、右に操作。
      - 取引詳細（出品者・取引中）は、左に受け渡しQR、右に完了リンク・完了コードの発行。
      - 受け渡し完了後は、左に完了の表示、右に商品と売上管理への導線。
      - 購入希望（決済モック）は、下部の合計と送信ボタンを右カラムにそろえる。
    - QR読み取りは、暗い地を列いっぱいにし、中身を 480px（640px 以上）に収める。
  - マイページ（1024px 以上）：左（320px）にプロフィール・設定のリンク（1列）、右にタブと出品・購入の履歴を置き、プロフィールとタブの上端をそろえる。見出しは上部ナビと重なるので読み上げにだけ残す。タブは文字の幅で左に寄せる。状態のチップと並び順は1行にまとめ、履歴の行は幅 320px 以上を保てる列数（1024px で1列、1280px で2列）にする。DOM の順はスマホと同じ。
- **タップ領域は 44px 以上**：アイコンボタン 44×44、主要ボタンの高さ 52px、副ボタン 48px、チップ 36px（行全体で44pxを確保）。
- **アクセシビリティ**：アイコンだけのボタンには `aria-label`、入力欄には `<label>`、エラーは `role="alert"`、トーストは `role="status"`、モーダルとボトムシートは `role="dialog"` と `aria-labelledby`。
- **絵文字は使わず**、アイコンは線画のインラインSVG（stroke 1.8〜2.4、24×24 viewBox）。
- 画面内の `[担当教員名]`、`[ISBN]`、`[料率]`、`[ドメイン]` などの角括弧はデザイン上のプレースホルダ。実データに置き換える。

## DS-2 デザイントークン

実装では [app/CLAUDE.md](../../app/CLAUDE.md) のとおり `app/src/styles/tokens.css` に **この表と同じ名前の CSS 変数** で定義する。表にない色を使いたくなったら、まずこの表に追加する（[spec-update スキル](../../.claude/skills/spec-update/SKILL.md)）。

### 色

| CSS変数 | 値 | 用途（デザインでの使われ方） |
| --- | --- | --- |
| `--color-bg` | `#F8F5EE` | 画面の地（一覧・検索・購入希望）、情報ボックスの地、検索欄の地 |
| `--color-surface` | `#FFFFFF` | カード、ヘッダー、ボトムナビ、フォーム画面の地 |
| `--color-text` | `#1B2B4B` | 本文・見出し（紺） |
| `--color-text-muted` | `#566178` | 補足文、ラベル、未選択のナビ |
| `--color-text-disabled` | `#8792A8` | 無効ボタンの文字（コード再送のカウントダウンなど） |
| `--color-border` | `#E4DFD3` | カード・入力欄・区切り線 |
| `--color-border-strong` | `#CFC8B8` | 未完了ステップの丸枠 |
| `--color-primary` | `#1F7A80` | 主要ボタン、選択中のチップ・ナビ、フォーカス中の入力枠（1.5px）、リンク |
| `--color-primary-hover` | `#155C61` | リンクのホバー、淡色バッジの文字 |
| `--color-primary-soft` | `#DDF0F0` | 淡色バッジ・選択チップの地、教科書カテゴリの写真プレースホルダ |
| `--color-primary-soft-2` | `#D4EEEE` | 淡色の地（マイページ） |
| `--color-primary-border` | `#B9DCDD` | カテゴリ別追加項目の入力枠（出品画面） |
| `--color-primary-dashed` | `#9CC9CB` | 写真追加ボタンの点線枠、カルーセルの非選択ドット |
| `--color-warning-bg` | `#FBEFC4` | 注意ボックス・「購入希望 n件」「決済はモック」バッジの地 |
| `--color-warning-text` | `#6B4E00` | 注意バッジの文字 |
| `--color-warning-text-strong` | `#4A3600` | 注意ボックスの本文 |
| `--color-warning-accent` | `#C99A12` | 審査中ステップの丸枠 |
| `--color-warning-badge` | `#F2C94C` | 管理画面の件数バッジ |
| `--color-info-bg` | `#DCEEF6` | 案内ボックスの地（再確認の理由）、家電カテゴリの写真プレースホルダ |
| `--color-info-text` | `#1B3A55` | 案内ボックスの文字 |
| `--color-danger` | `#A8322D` | 入力エラーの文字と枠 |
| `--color-danger-bg` | `#FBE7E4` | ロック中などの警告ボックスの地 |
| `--color-danger-text` | `#8A2621` | 警告ボックスの文字 |
| `--color-dark` | `#16223A` | QR読み取り画面（カメラ）の地 |
| `--color-dark-muted` | `#D3DAE6` | 暗い地の上の補足文 |
| `--color-admin-nav` | `#1B2B4B` | 管理画面サイドバーの地（`--color-text` と同値） |
| `--color-admin-nav-active` | `#2A3D63` | 管理画面サイドバーの選択中項目 |
| `--color-admin-nav-text` | `#C3CCDD` | 管理画面サイドバーの未選択項目 |
| `--color-overlay` | `rgba(27, 43, 75, 0.55)` | ボトムシート背面の幕 |
| `--color-overlay-dark` | `rgba(0, 0, 0, 0.5)` | カメラ画面上のモーダル背面の幕 |

**カテゴリの写真プレースホルダ色**（画像がないときの地）：教科書 `#DDF0F0`／家電 `#DCEEF6`／家具 `#FBF0CC`／ゲーム `#E1EFE3`。`--color-cat-textbook` `--color-cat-appliance` `--color-cat-furniture` `--color-cat-game` として定義する。商品のカテゴリ（`ItemCategory`）との対応は 教科書・参考書→教科書、家具・家電→家具、ゲーム・その他→ゲーム（家電の色は商品の写真には使わない。[ADR 0019](../decisions/0019-browse-category-chips-and-filter.md)）。

使われている箇所が1つだけの色（`#EFEBE2` `#1F3550` `#2F6FA3` など）は、実装時に [screens/](screens/) の該当箇所を見て上表のどれかに寄せるか、必要なら追加する。

### タイポグラフィ

| CSS変数 | 値 | 用途 |
| --- | --- | --- |
| `--font-body` | `'Zen Kaku Gothic New', sans-serif`（400/500/700） | 本文・見出しすべて |
| `--font-num` | `Outfit, sans-serif`（500/700） | 価格、学籍番号、メールアドレス、コード、カウントダウン、STEP表記、ISBN |

Google Fonts から `Zen+Kaku+Gothic+New:wght@400;500;700` と `Outfit:wght@500;700` を読み込む。

文字サイズは次の CSS 変数を使う（役割ごとの名前。同じ役割なら画面が違っても同じ変数を使う）。

| CSS変数 | 値 | 用途 |
| --- | --- | --- |
| `--text-badge` | `11px` | バッジ・ナビ・カードの補足 |
| `--text-note` | `12px` | 小さい注記・入力欄の補足 |
| `--text-label` | `13px` | 補足・ラベル・エラー文・チップ |
| `--text-body` | `14px` | 本文・テキストボタン・小さいボタン |
| `--text-button-md` | `15px` | 副ボタン（48px）・固定表示のドメイン |
| `--text-button` | `16px` | 主要ボタン（52px）・セクション見出し |
| `--text-subheading` | `17px` | ヘッダーのタイトル・入力欄の文字・一覧の価格 |
| `--text-header` | `18px` | 一覧系画面のヘッダー・シートの見出し |
| `--text-heading` | `20px` | 商品詳細のタイトル・ダイアログの見出し（DealScan の 19px もこれに寄せる） |
| `--text-code` | `22px` | コード入力の1桁・学籍番号の大きい表示・管理画面の見出し |
| `--text-title` | `24px` | フォーム系の画面タイトル |
| `--text-catch` | `26px` | ログインのキャッチ |
| `--text-price-lg` | `28px` | 商品詳細の価格 |
| `--text-display` | `30px` | 受け渡しコード（DealQR） |
| `--text-display-lg` | `34px` | 完了コード・売上の合計 |

| 用途 | サイズ / 太さ |
| --- | --- |
| 画面タイトル（フォーム系） | 24px / 700（ログインのキャッチは26px） |
| 商品詳細タイトル | 20px / 700 |
| ヘッダーのタイトル | 17〜18px / 700 |
| セクション見出し | 16px / 700 |
| 本文 | 14px（行間 1.7〜1.8） |
| 補足・ラベル | 13px（ラベルは700） |
| 小さい注記 | 12px |
| バッジ・ナビ | 11px |
| 価格（詳細 / 一覧 / 行） | Outfit 28px / 17px / 17〜18px、700 |
| 入力欄の文字 | 17px（コード入力は Outfit 22px） |

### 角丸・余白・影

| CSS変数 | 値 | 用途 |
| --- | --- | --- |
| `--radius-sm` | `8px` | コード入力の1桁、サムネイル |
| `--radius-input` | `10px` | テキスト入力 |
| `--radius-card` | `12px` | カード、情報ボックス |
| `--radius-dialog` | `18px` | 中央モーダル |
| `--radius-sheet` | `20px` | ボトムシート上端、QRカメラ枠 |
| `--radius-pill` | `999px` | ボタン（高さの半分 26/24/22/18px）、チップ、バッジ |
| `--space-page` | `16px` | 一覧系画面の左右余白（フォーム系は 24px = `--space-page-form`） |
| `--space-gap` | `12px` | カード間・要素間の基本間隔 |
| `--tap-min` | `44px` | 最小タップ領域 |
| `--btn-h` | `52px` | 主要ボタンの高さ |

影は使わず、`1px solid var(--color-border)` の枠線で面を分ける。

## DS-3 共通部品

デザインで繰り返し出てくる部品。実装時は `app/src/components/` に1つずつ作り、画面ごとに作り直さない。実装済みの部品と状態ごとの見た目は Storybook（`cd app && npm run storybook`）で確認できる（[ADR 0004](../decisions/0004-storybook-and-bottom-nav.md)）。全 story の描画と DS-1 のアクセシビリティは PR の CI で検査する（[ADR 0005](../decisions/0005-storybook-tests-in-ci.md)）。

| 部品 | 見た目 | 出てくる画面 |
| --- | --- | --- |
| `BottomNav` | 下部5タブ（ホーム／検索／出品／取引／マイページ）。選択中は primary + 700。PC 幅（1024px 以上）では上部の横ナビ（`placement="top"`。高さ56px、左にロゴ32px＋アプリ名、右に同じ5項目をアイコンとラベルの横並びで。DS-1、[ADR 0048](../decisions/0048-user-layout-pc-width.md)） | Main, Search, DealList, MyPage |
| `AppHeader` | 戻るボタン（44px）＋タイトル＋右側の補足（バッジ等）。高さ56px | 多数 |
| `Button` | primary（塗り）/ secondary（primary枠 1.5px）/ ghost（border枠）/ danger（danger枠 1.5px、キャンセル申請など）/ text、高さ 52・48・44。リンク版は `ButtonLink` | 全体 |
| `TextField` | ラベル＋高さ52pxの入力枠。フォーカス中は primary 1.5px、エラーは danger 1.5px | Login, SignUp, Password, Sell |
| `EmailField` | 学籍番号8桁の入力＋固定の `@edu.cc.saga-u.ac.jp` 表示 | Login, SignUp |
| `CodeInput` | 6桁×1マス（grid 6列、高さ52px、Outfit 22px）＋有効期限・再送 | Code, Recheck |
| `Chip` | 高さ36px。選択中 primary 塗り／絞り込み条件は primary-soft＋×アイコン | Main, Search, SearchFilter |
| `Badge` | 11〜12px / 700 / pill。primary-soft・warning を基本に、danger（停止など）・neutral（終了）・surface（写真の上のカテゴリ） | Detail, Request ほか |
| `ItemCard` | 正方形写真＋カテゴリバッジ＋タイトル2行＋価格＋キャンパス・状態（2列グリッド） | Main, MyPage |
| `ItemRow` | 84pxサムネイル＋タイトル＋授業名＋価格＋状態バッジ（横長） | Search |
| `InfoBox` | 角丸12px、アイコン＋文。info / warning / danger / neutral(bg) | Code, Recheck, Login, Password |
| `DefinitionList` | 2列グリッドの `<dl>`（項目名は muted） | Detail, Password |
| `BottomSheet` | 上端角丸20px＋つまみ＋背面の幕。見出しの右に操作（絞り込みの「クリア」）を置ける | DetailShare, DealCancel, SearchFilter, DealConfirm（つまみなし） |
| `ConfirmDialog` | 中央モーダル、角丸18px、主要ボタン＋テキストボタン。テキストボタンを省いた確定ボタンだけの形は Esc・背面の幕で閉じない（試験運用の初回ダイアログ。デザインは後で足す。[ADR 0039](../decisions/0039-trial-notice-dialog.md)） | DealScan, DealCompleteEnter |
| `SegmentedTabs` | 2択の切り替え（pill の枠内に高さ36pxのタブ2つ。選択中は primary 塗り）、`role="tablist"`。下線版（`variant="underline"`）は高さ44px・選択中に primary の下線3px＋700（[ADR 0030](../decisions/0030-mypage-history.md)） | DealList, AdminDeals, MyPage（下線版） |
| `Toast` | 紺地・白文字・pill、`role="status"` | DetailShare |
| `StepIndicator` | `STEP n / 3`（Outfit 13px / primary） | SignUp, Code, Password |

DS-3 の部品を作るために足した補助の部品（画面からも使える）：`Icon`（デザインの線画アイコンを名前で呼ぶ）、`ItemPhoto`（ItemCard・ItemRow の写真枠。写真がないときは `--color-cat-*` の地）、`Modal`（BottomSheet・ConfirmDialog の土台。幕・Esc・フォーカスの閉じ込めと復帰）、`SelectField`・`TextAreaField`（Sell のカテゴリ・状態の選択と説明。TextField と同じ枠・ラベル・エラーの形）、`Avatar`（学部の頭文字の丸いアイコン。Detail の出品者、DealRequests の購入者は 40px、MyPage の本人は 48px の白地（`size="lg"`・`inverse`））、`TwoColumn`（PC 幅で中身を左右2カラムに分ける枠。スマホでは枠を持たず縦に並ぶ。DS-1、[ADR 0048](../decisions/0048-user-layout-pc-width.md)）。

## DS-4 画面 ↔ アートボード ↔ 要件 の対応

| 画面（[08-screens](../product/08-screens.md)） | アートボード（[screens/](screens/)） | バリエーション（`data-props`） | 要件・技術 |
| --- | --- | --- | --- |
| ログイン・登録 | [Login](screens/Login.dc.html) | `locked`：10回失敗のロック表示 | [AUTH-1](../product/01-account-auth.md), [AUTHN-3・6](../tech/auth.md) |
| 〃 | [SignUp](screens/SignUp.dc.html) | `mode`：新規登録／パスワード再設定。実装では、新規登録のときだけ送信ボタンの上にデザインにない規約への同意（`TermsConsent`：「利用規約」「プライバシーポリシー」の文字リンクで本文を BottomSheet に開く。両方を開くまで「利用規約とプライバシーポリシーを確認し、同意します」のチェックボックスは無効で、下に「両方の文書を開くとチェックできます」を出す。チェックするまで送信ボタンは無効）を置く（[ADR 0040](../decisions/0040-terms-consent-and-mypage-links.md)。デザインは後で直す） | AUTH-1・2, AUTHN-2・4 |
| 〃 | [Code](screens/Code.dc.html) | `mode`：新規登録／再設定／再確認、`error` | AUTHN-1・2・6 |
| 〃 | [Password](screens/Password.dc.html) | `mode`：新規登録／再設定 | AUTHN-2・4・5 |
| 〃 | [Recheck](screens/Recheck.dc.html) | `reason`：30日経過／はじめての端末、`error` | AUTH-1, AUTHN-3 |
| 商品一覧 | [Main](screens/Main.dc.html) | `accent`（色の検討用。実装は primary 固定）。カテゴリチップは「すべて」＋要件の3区分、右上のお知らせ（ベル）は置かない（[ADR 0019](../decisions/0019-browse-category-chips-and-filter.md)） | [BROWSE](../product/04-browse-search-share.md) |
| 検索結果 | [Search](screens/Search.dc.html) | — | BROWSE |
| 〃（絞り込み） | [SearchFilter](screens/SearchFilter.dc.html) | `error`：価格の下限が上限より大きい | BROWSE |
| 商品詳細 | [Detail](screens/Detail.dc.html) | `shareApp`：右上の共有アイコン（既定オフ。実装しない。[ADR 0020](../decisions/0020-share-link-login-and-copy-only.md)）。実装では、売り切れ（取引中・取引完了）のとき「購入希望 n件」の位置に neutral の「売り切れ」バッジを出し、主ボタンを無効にする（[ADR 0018](../decisions/0018-browse-visible-items.md)） | BROWSE, [ITEM](../product/03-listing.md), [DEAL-1](../product/05-deal-flow.md) |
| 商品詳細（共有） | [DetailShare](screens/DetailShare.dc.html) | `copied`：コピー完了トースト、`shareApp`：「ほかのアプリで共有」ボタン（既定オフ。実装しない。ADR 0020） | BROWSE, [FE](../tech/frontend.md) |
| 購入希望・決済（モック） | [Request](screens/Request.dc.html) | — | DEAL-1, [PAY](../product/06-payment.md) |
| 取引詳細 | [DealRequests](screens/DealRequests.dc.html) | — | DEAL-1 |
| 〃 | [DealConfirm](screens/DealConfirm.dc.html) | — | DEAL-1・2 |
| 〃 | [DealQR](screens/DealQR.dc.html) | — | DEAL-3・4 |
| 〃 | [DealScan](screens/DealScan.dc.html) | — | DEAL-3 |
| 〃 | [DealCancel](screens/DealCancel.dc.html) | `side`：申請する／同意を求められた／申請中。実装では、相手の申請が届いていたらシートを一度だけ自動で開き、閉じた後は上部の warning の案内（「相手からキャンセルの申請が届いています。同意しなければ取引は続きます。」＋「内容を確認する」）から開き直せるようにする。キャンセル済みの取引詳細は「この取引はキャンセルされました。代金は購入者に返金されました。」 | DEAL-4, [ADR 0026](../decisions/0026-cancel-request-api.md) |
| 〃 | [DealCompleteIssue](screens/DealCompleteIssue.dc.html) | `kind`：完了リンク／完了コード（出品者） | DEAL-3 |
| 〃 | [DealCompleteEnter](screens/DealCompleteEnter.dc.html) | `via`：コード入力／完了リンク、`state`：入力／受け取り確認／エラー（購入者） | DEAL-3 |
| 取引一覧 | [DealList](screens/DealList.dc.html) | `tab`：購入／出品、`empty` | DEAL-1・3・4 |
| 出品・編集 | [Sell](screens/Sell.dc.html)（高さ1040） | — | ITEM, PAY-2, [STORAGE](../tech/storage.md) |
| 学生証登録 | [StudentCard](screens/StudentCard.dc.html) | — | [CARD](../product/02-student-card.md) |
| マイページ | [MyPage](screens/MyPage.dc.html) | 実装では、購入した商品は取引中・完了だけを出し、出品の行はデザインにない状態（公開中・取り下げ・停止中・売却済み）も出す（[ADR 0030](../decisions/0030-mypage-history.md)）。タブの下に、デザインにない状態の絞り込み（Chip）と並べ替え（新しい順／古い順の SelectField）を置く（[ADR 0031](../decisions/0031-mypage-filter-and-sort.md)。デザインは後で直す）。設定の一覧の「学生証の登録」の下に、デザインにない「利用規約」「プライバシーポリシー」の行を同じ形で足す（[ADR 0040](../decisions/0040-terms-consent-and-mypage-links.md)。デザインは後で直す） | AUTH-1・3, ITEM |
| 売上管理 | [Sales](screens/Sales.dc.html) | 実装では、明細の行に取引成立時の料率を「¥1,550 − 手数料（5%） ¥77」の形で出し、下の注記「手数料率は仮の値（10%）で表示しています。」は出さない。明細の行は商品詳細へのリンクにする。「出金を申請する」は ConfirmDialog で確認してから申請する。出金できる売上が 0 円なら押せない（[ADR 0032](../decisions/0032-sales-screen-note-and-payout-confirm.md)。デザインは後で直す） | PAY-2, [JOBS-2](../tech/jobs-payment-mock.md) |
| 管理：学生証審査 | [AdminCard](screens/AdminCard.dc.html)（PC 1280幅・流体） | — | CARD, [ADMIN](../product/07-admin.md) |
| 管理：ユーザー | [AdminUsers](screens/AdminUsers.dc.html)（PC 1280幅・流体） | — | ADMIN, AUTH-3 |
| 管理：出品・取引 | [AdminDeals](screens/AdminDeals.dc.html)（PC 1280幅・流体） | `tab`：出品／取引 | ADMIN, DEAL-4 |
| 管理：設定 | [AdminSettings](screens/AdminSettings.dc.html)（PC 1280幅・流体） | `error` | ADMIN, PAY-2 |

- デザインがない画面・状態は、既存画面の部品とトークンを組み合わせて作る。大きな画面を新しく起こすときは、先にユーザーへ確認する。
- 画面遷移（`href="X.dc.html"`）はデザイン上の仮のつながり。実際のルーティングは [tech/frontend.md](../tech/frontend.md) と `app/` の実装を正とする。
