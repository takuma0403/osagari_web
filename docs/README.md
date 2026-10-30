# ドキュメント索引

佐賀大お下がりフリマの仕様の入口です。**エージェントは作業の最初にこのファイルを読み、下の対応表から必要なファイルだけを開いてください。** 全部を読む必要はありません。

## 文書の地図

| ディレクトリ | 内容 | 正本か |
| --- | --- | --- |
| [product/](product/00-overview.md) | 要件（何を作るか）。機能ごとに分割 | 正本 |
| [tech/](tech/00-overview.md) | 技術構成（どう作るか）。分野ごとに分割 | 正本 |
| [design/](design/README.md) | デザイントークン・共通部品・画面の写し（`.dc.html`） | 正本は Claude Design のキャンバス、写しはリポジトリ |
| [api/openapi.yaml](api/openapi.yaml) | app ⇔ server の API 契約 | 正本 |
| [glossary.md](glossary.md) | 用語集（日本語 ↔ コード上の名前） | 正本 |
| [decisions/](decisions/0000-template.md) | 設計判断の記録（ADR） | 記録 |
| [product/open-questions.md](product/open-questions.md) / [tech/open-questions.md](tech/open-questions.md) | 未決事項。**ここに関わる作業は止めて質問する** | 正本 |
| [legal/](legal/terms.md) | 利用規約・プライバシーポリシーの本文 | 正本 |
| `_source/` | 分割前の元文書（2026-10-14時点で凍結）。**通常は読まない** | 記録 |

## 章ID

見出しの先頭に付いている `AUTH-1` `DEAL-3` などが章IDです。コミットメッセージ、PR、コードコメント、作業ブリーフで仕様を指すときはこのIDを使います（例：`// DEAL-3: トークンは一度使ったら無効にする`）。

- IDは振り直さない。節を追加するときは、その接頭辞の続きの番号を使う。
- 接頭辞の意味：`OVERVIEW` 概要／`AUTH` 認証・アカウント（要件）／`CARD` 学生証／`ITEM` 出品／`BROWSE` 一覧・検索・共有／`DEAL` 取引／`PAY` 決済／`ADMIN` 管理者／`SCREEN` 画面一覧／`DATA` データモデル／`OPEN` 未決（要件）／`ARCH` 構成／`COST` 費用／`FE` フロント／`API` API／`AUTHN` 認証（技術）／`DB` DB／`STORAGE` ファイル保存／`JOBS` 定期処理・決済モック／`TECH-OPEN` 未決（技術）／`DS` デザイン

## 機能 × 仕様 の対応表

| 機能 | 要件 | 技術 | 画面（[design](design/README.md)） | データ | 関連する未決事項 |
| --- | --- | --- | --- | --- | --- |
| 新規登録・ログイン・再確認・パスワード再設定 | [AUTH-1](product/01-account-auth.md) | [AUTHN-1〜7](tech/auth.md), [API](tech/api.md)（`__session`） | Login, SignUp, Code, Password, Recheck | User, セッション, 認証コード, 端末 | 教職員の利用 |
| 学籍番号からの学部・有効期限の推定 | [AUTH-2〜4](product/01-account-auth.md) | [AUTHN-2](tech/auth.md) | SignUp, Password, MyPage | User | 学部コードの最新版、大学院、延長 |
| アカウントの失効・削除 | [AUTH-3](product/01-account-auth.md) | [JOBS-1](tech/jobs-payment-mock.md) | MyPage | User, StudentCard | 問い合わせの窓口 |
| 学生証登録・審査 | [CARD](product/02-student-card.md), [ADMIN](product/07-admin.md) | [STORAGE](tech/storage.md), [JOBS-1](tech/jobs-payment-mock.md) | StudentCard, AdminCard | StudentCard | — |
| 出品・編集・取り下げ | [ITEM](product/03-listing.md) | [STORAGE](tech/storage.md) | Sell, MyPage | Item, ItemImage | 禁止出品物、取引中の取り下げ |
| 一覧・検索 | [BROWSE](product/04-browse-search-share.md) | [FE](tech/frontend.md), [DB](tech/database.md) | Main, Search | Item | — |
| 商品詳細・共有 | [BROWSE](product/04-browse-search-share.md) | [FE](tech/frontend.md)（共有文のコピー） | Detail, DetailShare | Item | — |
| 購入希望・承認・期限切れ | [DEAL-1](product/05-deal-flow.md) | [DB](tech/database.md)（条件付き更新）, [JOBS-1](tech/jobs-payment-mock.md) | Request, DealRequests, DealList | Transaction, Payment | 承認期限48hとオーソリ期限 |
| 取引成立の確認モーダル | [DEAL-2](product/05-deal-flow.md) | — | DealConfirm | Transaction | — |
| 受け渡し完了（QR・リンク・コード） | [DEAL-3](product/05-deal-flow.md) | [FE](tech/frontend.md)（QR）, [API](tech/api.md) | DealQR, DealScan, DealCompleteIssue, DealCompleteEnter | Transaction | — |
| キャンセル | [DEAL-4](product/05-deal-flow.md) | [JOBS-2](tech/jobs-payment-mock.md)（返金） | DealQR（申請ボタン）, DealCancel | Transaction, Payment | — |
| 自動完了（導入しない） | [DEAL-5](product/05-deal-flow.md) | — | — | — | — |
| 決済（モック）・手数料・売上 | [PAY](product/06-payment.md) | [JOBS-2](tech/jobs-payment-mock.md) | Request, Sales, Sell | Payment, Transaction | 決済サービス選定 |
| 管理者機能 | [ADMIN](product/07-admin.md) | [API](tech/api.md) | AdminCard, AdminUsers, AdminDeals, AdminSettings | 全般 | — |
| 定期処理・バックアップ | — | [JOBS-1](tech/jobs-payment-mock.md), [DB](tech/database.md) | — | — | — |
| 利用規約・プライバシーポリシー | [利用規約](legal/terms.md), [プライバシーポリシー](legal/privacy.md) | [FE](tech/frontend.md)（`/terms`・`/privacy`） | — | 全般 | 問い合わせの窓口、保存先の国（リージョン） |
| インフラ・費用 | [OVERVIEW](product/00-overview.md) | [ARCH・COST](tech/00-overview.md) | — | — | リージョン、独自ドメイン |

## 守るべき決まり（抜粋）

全文はルートの [CLAUDE.md](../CLAUDE.md) の「守るべき決まり」を参照。

## 運用ルール

- 仕様を変えるときは [spec-update スキル](../.claude/skills/spec-update/SKILL.md) に従い、分割後のファイルを直接直す。`_source/` は書き換えない。
- `node scripts/check-docs.mjs` でリンク切れと章IDの重複を検査する（docs を編集すると Hook で自動実行される）。
- `node scripts/check-docs.mjs --coverage` は分割直後の取りこぼし検査用。分割後に仕様を変更すると元文書と一致しなくなるため、通常は実行しない。
