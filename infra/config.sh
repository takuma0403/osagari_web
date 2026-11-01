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
# Cloud Scheduler のサービスアカウント（作成と権限付与は #40。内部用エンドポイントの照合に使う。JOBS-1）
SCHEDULER_SA="osagari-scheduler@${PROJECT_ID}.iam.gserviceaccount.com"

# コンテナイメージの置き場所
AR_REPO=osagari
IMAGE="${REGION}-docker.pkg.dev/${PROJECT_ID}/${AR_REPO}/api"

# Cloud Storage のバケット（#69, STORAGE）。常時無料枠のある us-central1 に置く（ADR 0041）
STORAGE_LOCATION=us-central1
BUCKET_ITEM_IMAGES="${PROJECT_ID}-item-images"
BUCKET_STUDENT_CARDS="${PROJECT_ID}-student-cards"
BUCKETS=("$BUCKET_ITEM_IMAGES" "$BUCKET_STUDENT_CARDS")
# ブラウザからの直接アップロードを CORS で許可するオリジン（Firebase Hosting の標準ドメイン。ADR 0041）
HOSTING_ORIGINS=("https://${PROJECT_ID}.web.app" "https://${PROJECT_ID}.firebaseapp.com")

# Secret Manager のシークレット名
SECRET_DATABASE_URL=database-url
SECRET_ADMIN_STUDENT_NUMBERS=admin-student-numbers
SECRETS=("$SECRET_DATABASE_URL" "$SECRET_ADMIN_STUDENT_NUMBERS")

# Turso の本番DB（グループは aws-ap-northeast-1。ADR 0041）
TURSO_DB=osagari
