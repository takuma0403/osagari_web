# infra（本番環境）

API（`server/`）を Cloud Run に出し、Turso と Secret Manager をつなぐためのスクリプトと手順です（#38）。方針は [ARCH・COST](../docs/tech/00-overview.md) と [ADR 0042](../docs/decisions/0042-cloud-run-secret-manager-deploy.md)、リージョンは [ADR 0041](../docs/decisions/0041-production-regions-and-domain.md)。

| ファイル | 役割 |
| --- | --- |
| [config.sh](config.sh) | プロジェクトID・リージョン・サービス名・シークレット名などの共通の値（秘密情報は書かない） |
| [setup.sh](setup.sh) | API の有効化、Artifact Registry、実行用サービスアカウント、シークレットの器と読み取り権限、Cloud Storage のバケット・CORS・権限 |
| [set-secret.sh](set-secret.sh) | 標準入力の値をシークレットの新しいバージョンにし、古いバージョンを破棄する |
| [deploy-server.sh](deploy-server.sh) | イメージのビルドと push → マイグレーション（Cloud Run ジョブ）→ Cloud Run サービスの更新 → `/api/health` の確認 |

スクリプトは `gcloud` に必ず `--project` を渡すので、手元の gcloud の既定プロジェクトは変えなくてよいです。どれも何度実行しても同じ結果になります。

## 本番環境の構成

| 要素 | 名前 | 備考 |
| --- | --- | --- |
| GCP プロジェクト | `osagari-web-dev` | Firebase も同じプロジェクト |
| Cloud Run サービス | `osagari-api`（asia-northeast1） | 0〜2台、1 vCPU・512MiB、CPU はリクエスト処理中だけ |
| Cloud Run ジョブ | `osagari-migrate` | 同じイメージの `/migrate up` |
| サービスアカウント | `osagari-api@…` | サービスとジョブの両方で使う |
| Artifact Registry | `osagari`（asia-northeast1） | 最新3件より古いイメージは自動で消える |
| Secret Manager | `database-url`・`admin-student-numbers` | 有効なバージョンは各1つ |
| Turso | `osagari`（default グループ、aws-ap-northeast-1） | |
| Cloud Storage | `osagari-web-dev-item-images`・`osagari-web-dev-student-cards`（us-central1） | 公開アクセス防止、ソフト削除なし。CORS は Firebase Hosting のオリジンからの PUT だけ許可 |

環境変数のうち秘密情報・個人情報（Turso の接続URL、管理者の学籍番号）は Secret Manager から、それ以外は `deploy-server.sh` が直接設定します。一覧は [server/.env.example](../server/.env.example)。

## 初回の構築

必要なもの：gcloud（請求先アカウントを使える Google アカウントでログイン）、turso CLI（ログイン済み）、docker。

1. プロジェクトを作り、請求先アカウントに紐づける。

   ```sh
   gcloud projects create osagari-web-dev
   gcloud billing accounts list
   gcloud billing projects link osagari-web-dev --billing-account <請求先アカウントID>
   ```

2. 土台を作る。

   ```sh
   infra/setup.sh
   ```

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

5. docker から Artifact Registry に push できるようにして、デプロイする。

   ```sh
   gcloud auth configure-docker asia-northeast1-docker.pkg.dev
   infra/deploy-server.sh
   ```

   最後に `{"status":"ok"}` と本番URL（`https://osagari-api-<プロジェクト番号>.asia-northeast1.run.app`）が表示されれば完了です。

## 日々の運用

- **デプロイ**：`infra/deploy-server.sh`。マイグレーションを先に適用してから API を更新します。イメージのタグは git のコミットで、`server/` に未コミットの変更があるときは止まります（どのコミットのイメージかを特定でき、ロールバック先を選べるようにするため）。
- **マイグレーションだけ**：`gcloud run jobs execute osagari-migrate --region asia-northeast1 --project osagari-web-dev --wait`。結果は Cloud Run ジョブのログで確認します。マイグレーションは前のバージョンの API でも動くように書きます（API より先に適用されるため）。
- **ロールバック**：前のリビジョンにトラフィックを戻します。スキーマは戻らないので、必要なら別にマイグレーションを書きます。

  ```sh
  gcloud run revisions list --service osagari-api --region asia-northeast1 --project osagari-web-dev
  gcloud run services update-traffic osagari-api --to-revisions <リビジョン名>=100 --region asia-northeast1 --project osagari-web-dev
  ```

- **Turso のトークンの入れ替え**：`turso db tokens invalidate osagari` で既存のトークンをすべて無効にし、手順3の2行目で新しいトークンを入れてから `infra/deploy-server.sh` を実行します。無効にしてから新しいリビジョンが出るまでは API が DB に接続できないので、利用の少ない時間に行います。
- **ログ**：`gcloud run services logs read osagari-api --region asia-northeast1 --project osagari-web-dev`

## 後続の issue で足すもの

- Cloud Scheduler とそのサービスアカウント `osagari-scheduler@…`、予算アラート、DBバックアップ（#40）
- Gmail の差出人とアプリパスワード（#41）。それまで `SMTP_FROM` は仮の値で、メール送信は失敗します
- Firebase Hosting と CI からのデプロイ（#39）。Hosting のサイト名が `osagari-web-dev` でなくなったら、`config.sh` の `HOSTING_ORIGINS` を直して `setup.sh` を実行し直す
