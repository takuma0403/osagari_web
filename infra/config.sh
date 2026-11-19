# 本番環境の共通の値（#38, ADR 0041, ADR 0042）。ほかのスクリプトが source する。
# 秘密情報はここに書かない（Secret Manager に置く）。

PROJECT_ID=osagari-web-dev
# Cloud Run・Artifact Registry は東京（ADR 0041）
REGION=asia-northeast1

# Cloud Run のサービスとマイグレーションのジョブ
SERVICE=osagari-api
MIGRATE_JOB=osagari-migrate
# API とマイグレーションを動かすサービスアカウント
RUNTIME_SA="osagari-api@${PROJECT_ID}.iam.gserviceaccount.com"
# Cloud Scheduler のサービスアカウント（#40）。内部用エンドポイントの照合に使う（JOBS-1, ADR 0027）
SCHEDULER_SA="osagari-scheduler@${PROJECT_ID}.iam.gserviceaccount.com"
# Cloud Scheduler のジョブ（#40, JOBS-1, ADR 0046）。無料枠に収めるため2つにまとめる。時刻は日本時間
SCHEDULER_LOCATION=$REGION
SCHEDULER_TIME_ZONE=Asia/Tokyo
SCHEDULER_JOB_SHORT=osagari-jobs-short
SCHEDULER_SCHEDULE_SHORT="*/15 * * * *"
SCHEDULER_JOB_DAILY=osagari-jobs-daily
SCHEDULER_SCHEDULE_DAILY="0 3 * * *"

# GitHub Actions からのデプロイ（#39, ADR 0045）。鍵ファイルを使わず、Workload Identity 連携で main の workflow だけがこの SA になれる
DEPLOYER_SA="osagari-deployer@${PROJECT_ID}.iam.gserviceaccount.com"
GITHUB_REPO=takuma0403/osagari_web
WIF_POOL=github
WIF_PROVIDER=github

# コンテナイメージの置き場所
AR_REPO=osagari
IMAGE="${REGION}-docker.pkg.dev/${PROJECT_ID}/${AR_REPO}/api"

# Cloud Storage のバケット（#69, STORAGE）。学生証だけを置き、常時無料枠のある us-central1 に置く（ADR 0041）。
# 商品写真は R2 に移した（#144, ADR 0051）
STORAGE_LOCATION=us-central1
BUCKET_STUDENT_CARDS="${PROJECT_ID}-student-cards"
BUCKETS=("$BUCKET_STUDENT_CARDS")
# 商品写真の Cloudflare R2（#141, STORAGE, ADR 0051）。佐賀から近いアジア太平洋の location hint を付ける。
# アカウントID は署名付きURLのホストに入り利用者のブラウザにも見えるので、ここに書く（アクセスキーは Secret Manager）
R2_ACCOUNT_ID=0d206bb093137deabf051d90b6d344f1
R2_LOCATION=apac
R2_BUCKET_ITEM_IMAGES=osagari-item-images
R2_ENDPOINT="https://${R2_ACCOUNT_ID}.r2.cloudflarestorage.com"
# setup.sh が使う wrangler の版（Node.js 22 以上が要る。リポジトリの .nvmrc の版で動かす）
WRANGLER_VERSION=4.156.2
# Firebase Hosting のサイト（既定のサイト名はプロジェクトIDと同じ。#39）。独自ドメインは使わない（ADR 0041）
HOSTING_SITE=$PROJECT_ID
# ブラウザからの直接アップロードを CORS で許可するオリジン（Firebase Hosting の標準ドメイン。ADR 0041）
HOSTING_ORIGINS=("https://${HOSTING_SITE}.web.app" "https://${HOSTING_SITE}.firebaseapp.com")
# deploy-app.sh が使う firebase-tools の版（CI と手元で揃える）
FIREBASE_TOOLS_VERSION=15.34.0

# Secret Manager のシークレット名
SECRET_DATABASE_URL=database-url
SECRET_ADMIN_STUDENT_NUMBERS=admin-student-numbers
# Gmail のアプリパスワード（#41, AUTHN-7）
SECRET_SMTP_PASSWORD=smtp-password
# 商品写真の R2 バケットだけを読み書きできる R2 API トークンのアクセスキー（#141, STORAGE）
SECRET_R2_ACCESS_KEY_ID=r2-access-key-id
SECRET_R2_SECRET_ACCESS_KEY=r2-secret-access-key
SECRETS=("$SECRET_DATABASE_URL" "$SECRET_ADMIN_STUDENT_NUMBERS" "$SECRET_SMTP_PASSWORD"
  "$SECRET_R2_ACCESS_KEY_ID" "$SECRET_R2_SECRET_ACCESS_KEY")

# 認証コードのメールを送るアプリ専用の Gmail アカウント（#41, AUTHN-7）。差出人として受信者に見えるので秘密ではない
SMTP_USERNAME=sagadai.osagari@gmail.com
SMTP_FROM="佐賀大お下がりフリマ 運営 <${SMTP_USERNAME}>"

# コードのメールの送信数が Gmail の上限に近づいたときの通知（#41, AUTHN-6）。通知先のアドレスは書かない（setup.sh の実行時に渡す）
ALERT_CHANNEL_NAME="osagari 運用通知"
ALERT_POLICY_NAME="コードのメールの送信数が上限に近い"

# 請求先アカウントの予算アラート（#40, COST, ADR 0046）。請求先アカウントの通貨が円であること
BUDGET_NAME="osagari 月の予算"
BUDGET_AMOUNT=500JPY
BUDGET_THRESHOLDS=(0.5 0.9 1.0)

# Turso の本番DB（グループは aws-ap-northeast-1。ADR 0041）
TURSO_DB=osagari

# R2 を使うスクリプトの最初に呼ぶ。アカウントID が未記入なら止める
require_r2_account_id() {
  if [ -z "$R2_ACCOUNT_ID" ]; then
    echo "infra/config.sh の R2_ACCOUNT_ID に Cloudflare のアカウントID を書いてください（infra/README.md）" >&2
    exit 1
  fi
}

# Cloud Run サービスの URL（https://<サービス名>-<プロジェクト番号>.<リージョン>.run.app）。
# 内部用エンドポイントの audience にも使う（deploy-server.sh の JOBS_OIDC_AUDIENCE と Cloud Scheduler のジョブで同じ値にする。JOBS-1）
service_url() {
  local number
  number=$(gcloud --project "$PROJECT_ID" projects describe "$PROJECT_ID" --format 'value(projectNumber)')
  echo "https://${SERVICE}-${number}.${REGION}.run.app"
}
