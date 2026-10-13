# デザイン（UIの正本）

- **キャンバス（正本）**: https://claude.ai/artifact/QdafhFt85mz3WvYPPtDGEH （Claude Design「佐賀大お下がりフリマ デザイン」）
- **リポジトリ内の写し**: [screens/](screens/) … キャンバスの各アートボードの `.dc.html` ソースと [canvas.json](screens/canvas.json)（2026-10-14 取得、version `1791906581-d01d`）
- **ロゴ**: [assets/logo.png](assets/logo.png)（1254×1254。画面ではアイコン枠に `object-fit: cover; transform: scale(1.32)` で表示している）
- 更新手順は [design-sync スキル](../../.claude/skills/design-sync/SKILL.md)、実装への移し方は [design-to-code スキル](../../.claude/skills/design-to-code/SKILL.md) を参照。

> `.dc.html` は Claude Design 独自のコンポーネント形式です（`<x-dc>` 内がマークアップ、`{{…}}` は末尾の `<script type="text/x-dc">` の `renderVals()` の値、`<sc-for>` は繰り返し、`<sc-if>` は条件表示、`data-props` は表示を切り替えるパラメータ）。**見た目と文言の参照元**として読み、コードとしてはそのまま流用しないこと。

## DS-1 基本方針

- **スマホ優先**：利用者向け画面は幅 390px（高さ 844px）のスマホを基準に作られている。管理画面だけはPC幅（1280px、流体レイアウト）。
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

**カテゴリの写真プレースホルダ色**（画像がないときの地）：教科書 `#DDF0F0`／家電 `#DCEEF6`／家具 `#FBF0CC`／ゲーム `#E1EFE3`。`--color-cat-textbook` `--color-cat-appliance` `--color-cat-furniture` `--color-cat-game` として定義する。

使われている箇所が1つだけの色（`#EFEBE2` `#1F3550` `#2F6FA3` など）は、実装時に [screens/](screens/) の該当箇所を見て上表のどれかに寄せるか、必要なら追加する。

### タイポグラフィ

| CSS変数 | 値 | 用途 |
| --- | --- | --- |
| `--font-body` | `'Zen Kaku Gothic New', sans-serif`（400/500/700） | 本文・見出しすべて |
| `--font-num` | `Outfit, sans-serif`（500/700） | 価格、学籍番号、メールアドレス、コード、カウントダウン、STEP表記、ISBN |

Google Fonts から `Zen+Kaku+Gothic+New:wght@400;500;700` と `Outfit:wght@500;700` を読み込む。

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

デザインで繰り返し出てくる部品。実装時は `app/src/components/` に1つずつ作り、画面ごとに作り直さない。

| 部品 | 見た目 | 出てくる画面 |
| --- | --- | --- |
| `BottomNav` | 下部5タブ（ホーム／検索／出品／取引／マイページ）。選択中は primary + 700 | Main, Search ほか |
| `AppHeader` | 戻るボタン（44px）＋タイトル＋右側の補足（バッジ等）。高さ56px | 多数 |
| `Button` | primary（塗り）/ secondary（primary枠 1.5px）/ ghost（border枠）/ text、高さ 52・48・44 | 全体 |
| `TextField` | ラベル＋高さ52pxの入力枠。フォーカス中は primary 1.5px、エラーは danger 1.5px | Login, SignUp, Password, Sell |
| `EmailField` | 学籍番号8桁の入力＋固定の `@edu.cc.saga-u.ac.jp` 表示 | Login, SignUp |
| `CodeInput` | 6桁×1マス（grid 6列、高さ52px、Outfit 22px）＋有効期限・再送 | Code, Recheck |
| `Chip` | 高さ36px。選択中 primary 塗り／絞り込み条件は primary-soft＋×アイコン | Main, Search |
| `Badge` | 11〜12px / 700 / pill。primary-soft・warning の2系統 | Detail, Request ほか |
| `ItemCard` | 正方形写真＋カテゴリバッジ＋タイトル2行＋価格＋キャンパス・状態（2列グリッド） | Main, MyPage |
| `ItemRow` | 84pxサムネイル＋タイトル＋授業名＋価格＋状態バッジ（横長） | Search |
| `InfoBox` | 角丸12px、アイコン＋文。info / warning / danger / neutral(bg) | Code, Recheck, Login, Password |
| `DefinitionList` | 2列グリッドの `<dl>`（項目名は muted） | Detail, Password |
| `BottomSheet` | 上端角丸20px＋つまみ＋背面の幕 | DetailShare |
| `ConfirmDialog` | 中央モーダル、角丸18px、主要ボタン＋テキストボタン | DealScan, DealConfirm |
| `Toast` | 紺地・白文字・pill、`role="status"` | DetailShare |
| `StepIndicator` | `STEP n / 3`（Outfit 13px / primary） | SignUp, Code, Password |

## DS-4 画面 ↔ アートボード ↔ 要件 の対応

| 画面（[08-screens](../product/08-screens.md)） | アートボード（[screens/](screens/)） | バリエーション（`data-props`） | 要件・技術 |
| --- | --- | --- | --- |
| ログイン・登録 | [Login](screens/Login.dc.html) | `locked`：10回失敗のロック表示 | [AUTH-1](../product/01-account-auth.md), [AUTHN-3・6](../tech/auth.md) |
| 〃 | [SignUp](screens/SignUp.dc.html) | `mode`：新規登録／パスワード再設定 | AUTH-1・2, AUTHN-2・4 |
| 〃 | [Code](screens/Code.dc.html) | `mode`：新規登録／再設定／再確認、`error` | AUTHN-1・2・6 |
| 〃 | [Password](screens/Password.dc.html) | `mode`：新規登録／再設定 | AUTHN-2・4・5 |
| 〃 | [Recheck](screens/Recheck.dc.html) | `reason`：30日経過／はじめての端末、`error` | AUTH-1, AUTHN-3 |
| 商品一覧 | [Main](screens/Main.dc.html) | `accent`（色の検討用。実装は primary 固定） | [BROWSE](../product/04-browse-search-share.md) |
| 検索結果 | [Search](screens/Search.dc.html) | — | BROWSE |
| 商品詳細 | [Detail](screens/Detail.dc.html) | — | BROWSE, [ITEM](../product/03-listing.md), [DEAL-1](../product/05-deal-flow.md) |
| 商品詳細（共有） | [DetailShare](screens/DetailShare.dc.html) | `copied`：コピー完了トースト | BROWSE, [FE](../tech/frontend.md) |
| 購入希望・決済（モック） | [Request](screens/Request.dc.html) | — | DEAL-1, [PAY](../product/06-payment.md) |
| 取引詳細 | [DealRequests](screens/DealRequests.dc.html) | — | DEAL-1 |
| 〃 | [DealConfirm](screens/DealConfirm.dc.html) | — | DEAL-1・2 |
| 〃 | [DealQR](screens/DealQR.dc.html) | — | DEAL-3・4 |
| 〃 | [DealScan](screens/DealScan.dc.html) | — | DEAL-3 |
| 出品・編集 | [Sell](screens/Sell.dc.html)（高さ1040） | — | ITEM, PAY-2, [STORAGE](../tech/storage.md) |
| 学生証登録 | [StudentCard](screens/StudentCard.dc.html) | — | [CARD](../product/02-student-card.md) |
| マイページ | [MyPage](screens/MyPage.dc.html) | — | AUTH-3, ITEM |
| 売上管理 | [Sales](screens/Sales.dc.html) | — | PAY-2, [JOBS-2](../tech/jobs-payment-mock.md) |
| 管理：学生証審査 | [AdminCard](screens/AdminCard.dc.html)（PC 1280幅・流体） | — | CARD, [ADMIN](../product/07-admin.md) |
| 管理：ユーザー | **未作成** | — | ADMIN |
| 管理：出品・取引 | **未作成** | — | ADMIN |
| 管理：設定 | **未作成** | — | ADMIN, PAY-2 |

- デザインがない画面（管理の3画面、取引詳細の「キャンセルの申請と同意」「完了リンク・コードの入力（購入者側）」など）は、既存画面の部品とトークンを組み合わせて作る。大きな画面を新しく起こすときは、先にユーザーへ確認する。
- 画面遷移（`href="X.dc.html"`）はデザイン上の仮のつながり。実際のルーティングは [tech/frontend.md](../tech/frontend.md) と `app/` の実装を正とする。
