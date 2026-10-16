# 用語集

仕様（日本語）とコード（英語）で同じものを同じ名前で呼ぶための対応表です。**新しい概念をコードに出すときは、先にここへ追加してください。** Go の型・TS の型・DB のテーブル名・OpenAPI のスキーマ名はこの表に揃えます。

> 関連: [データモデル](product/09-data-model.md) / [API契約](api/openapi.yaml)

## エンティティ

| 日本語 | コード（型） | テーブル | 説明 |
| --- | --- | --- | --- |
| ユーザー | `User` | `users` | 学生アカウント |
| 学生証 | `StudentCard` | `student_cards` | 出品に必要な学生証画像と審査状態 |
| 学生証の閲覧記録 | `StudentCardView` | `student_card_views` | 管理者が画像を閲覧した記録 |
| 商品 | `Item` | `items` | 出品物 |
| 商品画像 | `ItemImage` | `item_images` | 商品の写真（表示順つき） |
| 取引 | `Transaction` | `transactions` | 購入希望から完了までの1件。購入希望の段階も Transaction として作る |
| 購入希望 | `Transaction`（status=`requested`） | — | 独立したテーブルは作らない（[DATA](product/09-data-model.md)） |
| 受け渡しトークン | `HandoverToken` | `handover_tokens` | QR・完了リンク・完了コード |
| 決済・出金 | `Payment` | `payments` | 支払い・返金・出金の記録（モックでは外部IDが空） |
| セッション | `Session` | `sessions` | `__session` Cookie に入れるID |
| 認証コード | `VerificationCode` | `verification_codes` | メールで送る6桁コード（ハッシュで保存） |
| 登録端末 | `TrustedDevice` | `trusted_devices` | 再確認を省略できる端末 |
| パスワード設定用の一時トークン | `PasswordSetupToken` | `password_setup_tokens` | コードの確認後に発行する15分有効・一度だけのトークン（ハッシュで保存。[ADR 0007](decisions/0007-signup-flow.md)） |
| ログインチャレンジ | `LoginChallenge` | `login_challenges` | パスワードが一致したが再確認が必要なときに発行する、15分有効・一度だけのトークン。コードの確認・再送に使う（ハッシュで保存。[ADR 0009](decisions/0009-login-recheck-challenge.md)） |
| パスワードの入力ミスの記録 | — | `login_failures` | メールアドレスごとの入力ミスの回数とログインを止める期限（AUTHN-6。[ADR 0011](decisions/0011-rate-limits.md)） |
| 設定 | `Setting` | `settings` | 手数料率など |

## 列挙値

| 日本語 | コード | 値 |
| --- | --- | --- |
| ユーザーの状態 | `UserStatus` | `active` 有効 / `suspended` 停止 / `expired` 失効 / `deleted` 削除 |
| 出品者の承認状態 | `SellerStatus` | `none` 未登録 / `pending` 審査中 / `approved` 承認 / `rejected` 差し戻し |
| 学生証の審査状態 | `StudentCardStatus` | `pending` / `approved` / `rejected` |
| 商品のステータス | `ItemStatus` | `draft` 下書き / `published` 公開中 / `requested` 購入希望あり / `in_deal` 取引中 / `completed` 取引完了 / `withdrawn` 取り下げ / `suspended` 停止（管理者） |
| 取引のステータス | `TransactionStatus` | `requested` 購入希望 / `rejected` 却下 / `expired` 期限切れ / `in_deal` 取引中 / `cancelled` キャンセル / `completed` 完了 |
| 受け渡しの完了方法 | `HandoverMethod` | `qr` / `link` / `code` / `auto` |
| 受け渡しトークンの種類 | `HandoverTokenKind` | `qr` / `link` / `code` |
| 決済の種別 | `PaymentKind` | `authorize` 枠の確保 / `capture` 預かり / `release` 枠の解放 / `refund` 返金 / `settle` 売上の確定 / `payout` 出金 |
| 支払い方法 | `PaymentMethod` | `card` / `paypay` |
| カテゴリ | `ItemCategory` | `textbook` 教科書・参考書 / `furniture_appliance` 家具・家電 / `game_other` ゲーム・その他 |
| キャンパス | `Campus` | `honjo` 本庄 / `nabeshima` 鍋島 |
| 認証コードの用途 | `VerificationPurpose` | `signup` / `password_reset` / `login_recheck` |
| 一時トークンの用途 | `PasswordSetupPurpose` | `signup` 新規登録 / `password_reset` パスワードの再設定 |
| ログインの再確認の理由 | `LoginRecheckReason` | `code_expired` 最後のコード確認から30日以上 / `new_device` 初めての端末 |

- 商品の「状態」（目立った傷なし 等）はステータスと紛らわしいため、コードでは `condition`（`ItemCondition`）と呼ぶ。値の一覧はデザイン（[Sell](design/screens/Sell.dc.html)）と合わせて実装時に決める。
- デザインのカテゴリチップは「教科書・参考書／家具・家電／ゲーム／その他」と4つに分かれているが、要件（[ITEM](product/03-listing.md)）の区分は3つ。違いは実装時にユーザーへ確認する。

## そのほかの用語

| 日本語 | コード | 説明 |
| --- | --- | --- |
| 学籍番号 | `studentNumber` / `student_number` | メールアドレスの@より前（8桁） |
| 入学年度 | `admissionYear` | 学籍番号の上2桁 + 2000 |
| 学部・学科コード | `departmentCode` | 学籍番号の3〜5桁目 |
| 学部・学科 | `department` | [AUTH-4](product/01-account-auth.md) の表で確定した学部・学科のコード。表にないコードは空（不明）で登録し、管理者が後から直す。`departmentCode`（学籍番号から取った値）とは別に持つ |
| 有効期限 | `expiresOn` | アカウントの失効日（3月31日） |
| 学籍番号からの推定 | `StudentEstimate` | 学籍番号から推定した入学年度・学部・有効期限（[AUTH-2〜4](product/01-account-auth.md)） |
| 自分の情報 | `CurrentUser` | ログイン中の本人に返すユーザー情報（API のスキーマ名） |
| 手数料率 | `feeRate` | 取引ごとに記録する。整数のベーシスポイント（1% = 100）で保持し、浮動小数点を使わない |
| 手数料 | `fee` | 円（整数） |
| 出品者の売上 | `sellerProceeds` | 商品価格 − 手数料 |
| 承認期限 | `approvalDeadline` | 購入希望から48時間 |
| 共有用ID | `shareId` | 商品ページの固定URL `/items/:shareId` に使う |
| 確認モーダルへの同意日時 | `sellerAcknowledgedAt` | [DEAL-2](product/05-deal-flow.md) |
| 学籍番号の開示日時 | `identityDisclosedAt` | 承認時に記録 |
