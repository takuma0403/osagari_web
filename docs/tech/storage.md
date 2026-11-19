# ファイル保存（技術）

> 関連: [学生証登録（要件）](../product/02-student-card.md) / [出品・商品管理（要件）](../product/03-listing.md) / [定期処理](jobs-payment-mock.md)

## STORAGE ファイル保存

学生証は Cloud Storage、商品写真は Cloudflare R2 に保存します。商品写真は利用者が見るたびに読まれるので、転送料のかからない R2 に置きます（[ADR 0051](../decisions/0051-item-images-on-cloudflare-r2.md)）。どちらもブラウザから直接アップロードし、APIはアップロード用の署名付きURLを発行するだけにします。

| バケット | 保存先 | 公開範囲 | 閲覧方法 |
| --- | --- | --- | --- |
| 商品写真 | Cloudflare R2 | ログインした利用者のみ | APIが短時間有効な署名付きURLを発行する |
| 学生証 | Cloud Storage | 非公開 | 管理者向けAPIだけが署名付きURLを発行し、閲覧の記録を残す |

- **学生証の登録の流れ**：
  1. API がアップロード用の署名付きURL（PUT、有効期限15分、`Content-Type` は JPEG か WebP に固定）を発行する。オブジェクト名は `{ユーザーID}/{UUID}` にして、ほかの人の名前で登録できないようにする。署名には「まだないオブジェクトだけ作れる（`x-goog-if-generation-match: 0`）」「5MB まで（`x-goog-content-length-range`）」の条件も含め、登録した後に同じURLで画像を差し替えられないようにする。
  2. ブラウザがその URL に画像を直接 PUT する。
  3. ブラウザがオブジェクト名を API に送って登録する。API はオブジェクトが本人のものか、存在するか、形式が JPEG か WebP か、5MB 以下かを確かめてから審査待ちにする。
  4. 再提出したときは、前の学生証の行に置き換えた日時を入れて残し、画像も消さない（[ADR 0014](../decisions/0014-student-card-retention.md)）。本人にも画像の閲覧URLは出さない。
- **商品写真の登録の流れ**：学生証と同じく、API が PUT の署名付きURL（有効期限15分、JPEG か WebP、5MB まで、まだないオブジェクトだけ作れる）を `{ユーザーID}/{UUID}` の名前で発行し、ブラウザが直接 PUT する。R2 には `x-goog-content-length-range` に当たる条件がないため、ブラウザは発行を頼むときに縮小後の画像の大きさ（バイト）を送り、API は 5MB 以下かを確かめてから、その値を `Content-Length` として署名に含める。上書きの禁止は `If-None-Match: *` を署名に含めて行う（[ADR 0051](../decisions/0051-item-images-on-cloudflare-r2.md)）。出品・編集のときにオブジェクト名を並び順どおりに送り、API は本人のものか、存在するか、形式と大きさ、ほかの商品で使われていないかを確かめる（[ITEM](../product/03-listing.md)）。
- **商品写真の閲覧**：API が閲覧用の署名付きURL（GET、有効期限15分）を応答に入れる。閲覧の記録は残さない。
- **学生証の閲覧**：管理API が閲覧用の署名付きURL（GET、有効期限15分）を発行し、そのたびに閲覧した管理者と日時を `student_card_views` に記録する。記録は画像を削除した後も残す（[ADR 0015](../decisions/0015-admin-authorization.md)）。
- **ローカル**：学生証は fake-gcs-server、商品写真は S3 互換の MinIO を使う。MinIO は本番の R2 と同じく署名を検証するので、API は MinIO のアクセスキーで署名し、ブラウザから届くホストを URL に入れる。fake-gcs-server は署名を検証せずに署名付きURL形式の PUT を受け付け、CORS も許可する（署名に含まれる `x-goog-content-length-range`・`x-goog-if-generation-match` は `-cors-headers` で許可ヘッダーに加える）。API はダミーの署名で URL を作り、ブラウザから届くホスト（`STORAGE_PUBLIC_HOST`）を URL に入れる。
- **画像の縮小**：アップロード前にブラウザで縮小・圧縮し、保存容量と転送量を抑えます（例：長辺1600px、JPEGまたはWebP）。
- **学生証の削除**：アカウントの失効・削除から3年たった画像を、定期処理でDBの削除予定日に従って削除し、`image_deleted_at` を記録します（行は残す）。アップロードURLを発行したまま登録されなかったオブジェクトも、作られてから24時間たったら定期処理で削除します（[ADR 0033](../decisions/0033-account-expiry-and-daily-job.md)）。
- **暗号化**：Cloud Storage・R2 とも保存データを標準で暗号化しています。学生証のバケットには公開アクセス防止を設定し（[ADR 0043](../decisions/0043-production-cloud-storage-settings.md)）、商品写真の R2 バケットには公開用のURL（r2.dev・独自ドメイン）を付けず（[ADR 0051](../decisions/0051-item-images-on-cloudflare-r2.md)）、誤って公開されないようにします。
- **学生証の本番の設定**（Cloud Storage。[ADR 0043](../decisions/0043-production-cloud-storage-settings.md)、手順は [infra/README.md](../../infra/README.md)）：
  - ソフト削除は無効にし、削除したオブジェクトはすぐに消える（期限が来た学生証を確実に消すため）。
  - CORS は Firebase Hosting の標準ドメインからの PUT だけを許可し、`Content-Type`・`x-goog-if-generation-match`・`x-goog-content-length-range` を通す。
  - API のサービスアカウントには、バケットの読み書きの権限と、自分自身への `roles/iam.serviceAccountTokenCreator` を付ける。署名付きURLは鍵ファイルを使わず IAM signBlob で署名する。
- **商品写真の本番の設定**（Cloudflare R2。[ADR 0051](../decisions/0051-item-images-on-cloudflare-r2.md)、手順は [infra/README.md](../../infra/README.md)）：
  - R2 にはソフト削除がなく、削除したオブジェクトはすぐに消える。
  - CORS は Firebase Hosting の標準ドメインからの PUT だけを許可し、`Content-Type`・`If-None-Match` を通す（`Content-Length` はブラウザが付ける）。
  - API は R2 の S3 互換 API を使い、アクセスキーで署名する。アクセスキーは Secret Manager に保存し、権限は商品写真のバケットの読み書きだけにする。
  - 署名付きURLは R2 の S3 互換のエンドポイント（`<アカウントID>.r2.cloudflarestorage.com`）を指す。
- **リージョン**：学生証は、Cloud Storage の常時無料枠（5GB）の対象になる us-central1 に置きます（[ADR 0041](../decisions/0041-production-regions-and-domain.md)）。商品写真の R2 バケットには、佐賀から近いアジア太平洋（APAC）の location hint を付けます。保存する国は Cloudflare が決め、指定できません（[ADR 0051](../decisions/0051-item-images-on-cloudflare-r2.md)）。
