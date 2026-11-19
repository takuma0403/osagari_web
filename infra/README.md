# infra（本番環境）

API（`server/`）を Cloud Run に、画面（`app/`）を Firebase Hosting に出し、Turso と Secret Manager をつなぐためのスクリプトと手順です（#38, #39）。方針は [ARCH・COST](../docs/tech/00-overview.md) と [ADR 0042](../docs/decisions/0042-cloud-run-secret-manager-deploy.md)、リージョンは [ADR 0041](../docs/decisions/0041-production-regions-and-domain.md)、メールは [ADR 0044](../docs/decisions/0044-production-gmail-smtp.md)、Hosting と CI からのデプロイは [ADR 0045](../docs/decisions/0045-hosting-and-ci-deploy.md)、定期処理と予算アラートは [ADR 0046](../docs/decisions/0046-cloud-scheduler-and-budget-alert.md)、商品写真の Cloudflare R2 は [ADR 0051](../docs/decisions/0051-item-images-on-cloudflare-r2.md)。

| ファイル | 役割 |
| --- | --- |
| [config.sh](config.sh) | プロジェクトID・リージョン・サービス名・シークレット名などの共通の値（秘密情報は書かない） |
| [setup.sh](setup.sh) | API の有効化、Artifact Registry、実行用サービスアカウント、シークレットの器と読み取り権限、Cloud Storage の学生証のバケット・CORS・権限、Cloudflare R2 のバケット・CORS（公開URLは付けない）、メールの送信数の通知、GitHub Actions 用の Workload Identity とデプロイ用サービスアカウント、定期処理の Cloud Scheduler、予算アラート |
| [set-secret.sh](set-secret.sh) | 標準入力の値をシークレットの新しいバージョンにし、古いバージョンを破棄する |
| [create-r2-token.sh](create-r2-token.sh) | 商品写真の R2 バケットだけを読み書きできる R2 API トークンを作り、アクセスキーを Secret Manager に入れる。`prune` で古いトークンを消す |
| [deploy-server.sh](deploy-server.sh) | イメージのビルドと push → マイグレーション（Cloud Run ジョブ）→ Cloud Run サービスの更新 → `/api/health` の確認 |
| [deploy-app.sh](deploy-app.sh) | 画面のビルド → Firebase Hosting へのデプロイ（設定はルートの [firebase.json](../firebase.json)）→ Hosting 経由の `/api/health` の確認 |
| [../.github/workflows/deploy.yml](../.github/workflows/deploy.yml) | main への push で、変更のあった側を上の2つのスクリプトで出す（API → 画面の順）。R2 の接続先とアクセスキーは `deploy-server.sh` が渡すので、workflow に R2 の設定はない |

スクリプトは `gcloud` に必ず `--project` を、wrangler に `CLOUDFLARE_ACCOUNT_ID` を渡すので、手元の既定のプロジェクト・アカウントは変えなくてよいです。どれも何度実行しても同じ結果になります。

## 本番環境の構成

| 要素 | 名前 | 備考 |
| --- | --- | --- |
| GCP プロジェクト | `osagari-web-dev` | Firebase も同じプロジェクト |
| Cloud Run サービス | `osagari-api`（asia-northeast1） | 0〜2台、1 vCPU・512MiB、CPU はリクエスト処理中だけ |
| Cloud Run ジョブ | `osagari-migrate` | 同じイメージの `/migrate up` |
| Firebase Hosting | サイト `osagari-web-dev`（`https://osagari-web-dev.web.app`） | `/api/**` を `osagari-api` に転送し、ほかは `index.html` を返す |
| サービスアカウント | `osagari-api@…` | サービスとジョブの両方で使う |
| サービスアカウント | `osagari-scheduler@…` | Cloud Scheduler が内部用エンドポイントを呼ぶときの OIDC トークンに使う。ロールは付けない（[JOBS-1](../docs/tech/jobs-payment-mock.md)） |
| サービスアカウント | `osagari-deployer@…` | GitHub Actions がデプロイに使う。鍵は作らない |
| Workload Identity | プール `github`・プロバイダ `github` | `takuma0403/osagari_web` の main の workflow だけを受け入れる |
| Artifact Registry | `osagari`（asia-northeast1） | 最新3件より古いイメージは自動で消える |
| Secret Manager | `database-url`・`admin-student-numbers`・`smtp-password`・`r2-access-key-id`・`r2-secret-access-key` | 有効なバージョンは各1つ（合計5つ。無料枠は6つ） |
| Turso | `osagari`（default グループ、aws-ap-northeast-1） | |
| Cloud Storage | `osagari-web-dev-student-cards`（us-central1） | 学生証。公開範囲・CORS・権限は [STORAGE](../docs/tech/storage.md)。商品写真のバケット（`osagari-web-dev-item-images`）は R2 に移して消した（#144） |
| Cloudflare R2 | `osagari-item-images`（location hint APAC） | 商品写真。公開URL（r2.dev・独自ドメイン）は付けない。CORS は [STORAGE](../docs/tech/storage.md)。API は S3 互換のエンドポイント（`<アカウントID>.r2.cloudflarestorage.com`）に、このバケットだけを読み書きできる R2 API トークンのアクセスキーで接続する |
| Gmail | `sagadai.osagari@gmail.com` | 認証コードの送信元。2段階認証を有効にし、アプリパスワードで送る（[AUTHN-7](../docs/tech/auth.md)） |
| Cloud Scheduler | `osagari-jobs-short`（15分ごと）・`osagari-jobs-daily`（毎日3時）（asia-northeast1、Asia/Tokyo） | `POST /api/internal/jobs/short`・`/daily` を呼ぶ。失敗は1分以上あけて3回まで再試行 |
| 予算 | `osagari 月の予算`（請求先アカウント全体、月500円） | 50%・90%・100%で請求先アカウントの管理者と運用通知のチャンネルに通知（[COST](../docs/tech/00-overview.md)） |
| Cloud Monitoring | 通知チャンネル `osagari 運用通知`、アラートポリシー `コードのメールの送信数が上限に近い` | 直近24時間の送信が400通に達した警告ログで、1時間に1回まで運用者にメールする（[AUTHN-6](../docs/tech/auth.md)） |

環境変数のうち秘密情報・個人情報（Turso の接続URL、管理者の学籍番号、Gmail のアプリパスワード、R2 のアクセスキー）は Secret Manager から、それ以外は `deploy-server.sh` が直接設定します。一覧は [server/.env.example](../server/.env.example)。

## 初回の構築

必要なもの：gcloud（請求先アカウントを使える Google アカウントでログイン。予算の作成に請求先アカウントの管理者の権限が要る。`gcloud beta` のコマンドも使う）、turso CLI（ログイン済み）、docker、jq、送信用の Gmail アカウントのアプリパスワード、Cloudflare アカウント、Node.js（リポジトリの `.nvmrc` の版。wrangler を `npx` で動かす。`nvm use`）。

1. プロジェクトを作り、請求先アカウントに紐づける。

   ```sh
   gcloud projects create osagari-web-dev
   gcloud billing accounts list
   gcloud billing projects link osagari-web-dev --billing-account <請求先アカウントID>
   ```

   Cloudflare のアカウントを作り、ダッシュボードの「R2 Object Storage」で R2 を有効にする（無料枠だけで使う場合もカードの登録が要る。[ADR 0051](../docs/decisions/0051-item-images-on-cloudflare-r2.md)）。ダッシュボードに表示されるアカウントID を [config.sh](config.sh) の `R2_ACCOUNT_ID` に書いてコミットし（署名付きURLのホストに入り利用者にも見えるので秘密ではない）、wrangler にログインする。

   ```sh
   nvm use
   npx wrangler@<config.sh の WRANGLER_VERSION> login
   ```

2. 土台を作る。初回は、メールの送信数の通知を受け取る運用者のアドレスを `ALERT_EMAIL` で渡す（リポジトリには書かない）。通知チャンネルを作ったあとは、`ALERT_EMAIL` なしで実行できる。

   ```sh
   ALERT_EMAIL=<通知先のアドレス> infra/setup.sh
   ```

   通知が届くかは、Cloud Console の「Monitoring → アラート → 通知チャンネルを編集」からテスト通知を送って確かめる。

   予算の金額は円（`500JPY`）なので、請求先アカウントの通貨が円でないと作れない。Cloud Scheduler のジョブも作られるが、手順7で API を出すまでは呼び出しが失敗する（出したあとは次の回から通る）。

3. Turso の本番DBを作り、接続URL（トークン入り）を Secret Manager に入れる。値は画面に出さない。

   ```sh
   turso db create osagari --group default
   printf '%s?authToken=%s' "$(turso db show osagari --url)" "$(turso db tokens create osagari)" \
     | infra/set-secret.sh database-url
   ```

4. 管理者の学籍番号（8桁のカンマ区切り。Secret Manager は空の値を入れられないので、最低1人は入れる）を入れる。リポジトリには書かない。入力は画面に表示されない。

   ```sh
   read -rs ADMINS && printf '%s' "$ADMINS" | infra/set-secret.sh admin-student-numbers; unset ADMINS
   ```

5. Gmail のアプリパスワード（16文字。空白は除く）を入れる。入力は画面に表示されない。

   ```sh
   read -rs SMTP_PASSWORD && printf '%s' "$SMTP_PASSWORD" | infra/set-secret.sh smtp-password; unset SMTP_PASSWORD
   ```

6. 商品写真の R2 バケットのアクセスキーを入れる。`create-r2-token.sh` が、`osagari-item-images` のオブジェクトの読み書きだけができる R2 API トークンを Cloudflare API で作り、S3 互換 API のアクセスキー（ID はトークンの ID、シークレットはトークンの値の SHA-256）を Secret Manager に入れる。値は画面に出さない。

   Cloudflare API を呼ぶために、作業用のトークンを1本だけダッシュボードで作る。「プロフィール → API トークン」ではなく「アカウントの管理 → アカウント API トークン」で、カスタムトークンにし、権限は「アカウント → アカウント API トークン → 編集」だけ、TTL は当日までにする。スクリプトを実行すると入力を求められる（画面に表示されない）。終わったら作業用のトークンはダッシュボードで消す。

   ```sh
   infra/create-r2-token.sh
   ```

   `deploy-server.sh` はこの2つを Cloud Run に渡すので、値を入れる前に API をデプロイすると（main へのマージを含む）失敗する。

7. docker から Artifact Registry に push できるようにして、デプロイする。

   ```sh
   gcloud auth configure-docker asia-northeast1-docker.pkg.dev
   infra/deploy-server.sh
   ```

   最後に `{"status":"ok"}` と本番URL（`https://osagari-api-<プロジェクト番号>.asia-northeast1.run.app`）が表示されれば完了です。

8. 大学アドレスに認証コードのメールが届くか確かめる（AUTHN-7）。自分の大学アドレス宛てに新規登録のコードを送る。

   ```sh
   curl -fsS -X POST "<本番URL>/api/auth/signup/codes" -H 'Content-Type: application/json' \
     -d '{"email":"<学籍番号>@edu.cc.saga-u.ac.jp"}'
   ```

   - 受信トレイに届いたか、迷惑メールに振り分けられたかを見る。メールのヘッダー（`Authentication-Results`）で SPF・DKIM・DMARC が `pass` になっているかも見る。
   - 届かないときは、ログで送信のエラー（`mail:` で始まる）を探す。

   結果は [tech/open-questions.md](../docs/tech/open-questions.md) の「大学のメールでGmailからの送信が迷惑メール扱いされないかの確認」に記録する。

9. Firebase をプロジェクトに追加し（既定のサイト `osagari-web-dev` ができる）、`setup.sh` を実行し直して GitHub Actions 用の設定を作る。

   ```sh
   firebase login
   firebase projects:addfirebase osagari-web-dev
   infra/setup.sh
   ```

   `deploy.yml` の `WORKLOAD_IDENTITY_PROVIDER` はプロジェクト番号を含む。プロジェクトを作り直したら書き換える。

10. 画面を手元から出して、Hosting 経由で動くか確かめる。

   ```sh
   infra/deploy-app.sh
   ```

   - `https://osagari-web-dev.web.app/items/xxxx` を開いて画面が出る（SPA のフォールバック）
   - 新規登録 → ログインまで通り、マイページが開く（`__session` が Hosting 経由で往復する。[API](../docs/tech/api.md)）

   以降は dev → main のマージで GitHub Actions が出す。

## 日々の運用

- **デプロイ**：dev → main のマージで GitHub Actions（`deploy.yml`）が出します。dev への push では出ません。Actions の「Deploy」を main で手動実行すると、変更の有無にかかわらず両方を出します。手元から出すときは下の2つです。
- **API のデプロイ**：`infra/deploy-server.sh`。マイグレーションを先に適用してから API を更新します。イメージのタグは git のコミットで、`server/` に未コミットの変更があるときは止まります（どのコミットのイメージかを特定でき、ロールバック先を選べるようにするため）。
- **マイグレーションだけ**：`gcloud run jobs execute osagari-migrate --region asia-northeast1 --project osagari-web-dev --wait`。結果は Cloud Run ジョブのログで確認します。マイグレーションは前のバージョンの API でも動くように書きます（API より先に適用されるため）。
- **ロールバック**：前のリビジョンにトラフィックを戻します。スキーマは戻らないので、必要なら別にマイグレーションを書きます。

  ```sh
  gcloud run revisions list --service osagari-api --region asia-northeast1 --project osagari-web-dev
  gcloud run services update-traffic osagari-api --to-revisions <リビジョン名>=100 --region asia-northeast1 --project osagari-web-dev
  ```

- **Turso のトークンの入れ替え**：`turso db tokens invalidate osagari` で既存のトークンをすべて無効にし、手順3の2行目で新しいトークンを入れてから `infra/deploy-server.sh` を実行します。無効にしてから新しいリビジョンが出るまでは API が DB に接続できないので、利用の少ない時間に行います。
- **画面のデプロイ**：`infra/deploy-app.sh`。`app/` か `firebase.json` に未コミットの変更があるときは止まります。戻すときは Firebase Console の Hosting のリリース履歴から前のリリースを「ロールバック」します（リリースのメッセージがコミットです）。
- **ログ**：`gcloud run services logs read osagari-api --region asia-northeast1 --project osagari-web-dev`
- **定期処理**：状態（最後に動いた時刻と結果）は一覧で、すぐ動かしたいときは手動で実行します。ジョブの結果は API のログ（`/api/internal/jobs/*`）で見ます。認証に失敗すると 404 になります（audience かサービスアカウントが API の環境変数と合っていない）。

  ```sh
  gcloud scheduler jobs list --location asia-northeast1 --project osagari-web-dev
  gcloud scheduler jobs run osagari-jobs-short --location asia-northeast1 --project osagari-web-dev
  ```

- **時刻・再試行・予算の変更**：[config.sh](config.sh) と [setup.sh](setup.sh) を直して `infra/setup.sh` を実行し直します（ジョブと予算は上書きされます）。
- **Gmail のアプリパスワードの入れ替え**：Google アカウントで新しいアプリパスワードを作り、手順5で入れて `infra/deploy-server.sh` を実行してから、古いアプリパスワードを削除します。
- **R2 のアクセスキーの入れ替え**：手順6と同じく作業用のトークンを作り、`infra/create-r2-token.sh` で新しいトークンを作って入れ、`infra/deploy-server.sh` を実行してから、`infra/create-r2-token.sh prune` で古いトークンを消します（Secret Manager に入っているもの以外の、このスクリプトが作ったトークンを消す）。
- **通知先・アラートの変更**：`setup.sh` は通知チャンネルとアラートポリシーを表示名で探し、あれば作り直しません。通知先を変える、またはポリシーの中身を変えるときは、Cloud Console で直すか、消してから `setup.sh` を実行し直します。
- **送信数の通知が来たら**：無料の Gmail は1日およそ500通までです。ログで送信先の偏りや不正な利用がないかを確認します（[AUTHN-6](../docs/tech/auth.md)）。

## 記録：商品写真の Cloud Storage から R2 への移行（#144、2026-11-19 実施済み）

商品写真を Cloud Storage（`osagari-web-dev-item-images`）から R2（`osagari-item-images`）に移したときの手順です（[ADR 0051](../docs/decisions/0051-item-images-on-cloudflare-r2.md)）。オブジェクト名（`{ユーザーID}/{UUID}`）は同じなので DB は変えていません。同じような移し替えをするときの参考に残します。

1. 本番がまだ Cloud Storage を使っているうちに、rclone で Cloud Storage → R2 にコピーする。rclone は公式の Docker イメージで動かし、Cloud Storage へはアプリケーションのデフォルト認証情報（`gcloud auth application-default login`）、R2 へは Secret Manager にある API 用のアクセスキーで接続する（キーは環境変数で渡し、画面に出さない）。Content-Type は rclone が引き継ぐ。

   ```sh
   source infra/config.sh
   rcl() {
     RCLONE_S3_ACCESS_KEY_ID=$(gcloud --project "$PROJECT_ID" secrets versions access latest --secret "$SECRET_R2_ACCESS_KEY_ID") \
     RCLONE_S3_SECRET_ACCESS_KEY=$(gcloud --project "$PROJECT_ID" secrets versions access latest --secret "$SECRET_R2_SECRET_ACCESS_KEY") \
     docker run --rm \
       -v "$HOME/.config/gcloud/application_default_credentials.json:/adc.json:ro" \
       -e GOOGLE_APPLICATION_CREDENTIALS=/adc.json \
       -e RCLONE_GCS_ENV_AUTH=true -e RCLONE_GCS_BUCKET_POLICY_ONLY=true \
       -e RCLONE_S3_PROVIDER=Cloudflare -e RCLONE_S3_ENDPOINT="$R2_ENDPOINT" -e RCLONE_S3_NO_CHECK_BUCKET=true \
       -e RCLONE_S3_ACCESS_KEY_ID -e RCLONE_S3_SECRET_ACCESS_KEY \
       rclone/rclone:1.70.3 "$@"
   }
   rcl copy :gcs:osagari-web-dev-item-images ":s3:${R2_BUCKET_ITEM_IMAGES}" -v
   rcl check :gcs:osagari-web-dev-item-images ":s3:${R2_BUCKET_ITEM_IMAGES}" --one-way   # 0 differences found を確かめる
   ```

2. dev → main のマージで、R2 を使う API と画面をデプロイする（API → 画面の順）。
3. 手順1の `copy` と `check` をもう一度実行し、コピーからデプロイまでの間に Cloud Storage に上がった写真を R2 に移す。
4. 本番で、既存の商品の写真が表示されることと、新しい出品の写真を上げて表示できることを確かめる。
5. Cloud Storage のバケットを中身ごと消す（`gcloud storage rm -r gs://osagari-web-dev-item-images --project osagari-web-dev`。バケット単位の権限も一緒に消える）。`config.sh`・`setup.sh`・`deploy-server.sh` からバケットと環境変数 `STORAGE_BUCKET_ITEM_IMAGES` を外し、次のデプロイで Cloud Run の環境変数からも消す。

実施時は7件（約2.8MB）で、手順3で追加のコピーはなかった。

## 後続の issue で足すもの

- DBバックアップ（#118。方式は未定）
