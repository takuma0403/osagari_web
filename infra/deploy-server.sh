#!/usr/bin/env bash
# API を本番に出す（#38, ADR 0042）：イメージのビルドと push → マイグレーション（Cloud Run ジョブ）→ Cloud Run サービスの更新 → 稼働確認。
# 前提：infra/setup.sh と、シークレットの値の投入が済んでいること（infra/README.md）。
set -euo pipefail
cd "$(dirname "$0")"
source ./config.sh

gc() { gcloud --project "$PROJECT_ID" --quiet "$@"; }

repo_root=$(git rev-parse --show-toplevel)
tag=$(git rev-parse --short=12 HEAD)
if [ -n "$(git status --porcelain -- "$repo_root/server")" ]; then
  # タグからイメージの中身を特定できるよう（ロールバック先を選べるよう）、コミット済みのものだけを出す
  echo "server/ に未コミットの変更があります。コミットしてから実行してください" >&2
  exit 1
fi
image="${IMAGE}:${tag}"

echo "==> イメージをビルドして push する（${image}）"
docker build --target prod -t "$image" "$repo_root/server"
docker push "$image"

echo "==> マイグレーションを適用する（${MIGRATE_JOB}）"
gc run jobs deploy "$MIGRATE_JOB" --region "$REGION" --image "$image" \
  --service-account "$RUNTIME_SA" \
  --command /migrate --args up \
  --set-secrets "DATABASE_URL=${SECRET_DATABASE_URL}:latest" \
  --tasks 1 --max-retries 0 --task-timeout 300s
gc run jobs execute "$MIGRATE_JOB" --region "$REGION" --wait

# 内部用エンドポイントの audience にはサービスの URL を使う（Cloud Scheduler のジョブにも同じ値を設定する。JOBS-1, #40）
url=$(service_url)

env_file=$(mktemp)
trap 'rm -f "$env_file"' EXIT
cat >"$env_file" <<YAML
STORAGE_BUCKET_ITEM_IMAGES: "${BUCKET_ITEM_IMAGES}"
STORAGE_BUCKET_STUDENT_CARDS: "${BUCKET_STUDENT_CARDS}"
SMTP_HOST: "smtp.gmail.com"
SMTP_PORT: "587"
SMTP_USERNAME: "${SMTP_USERNAME}"
SMTP_FROM: "${SMTP_FROM}"
JOBS_OIDC_AUDIENCE: "${url}"
JOBS_SERVICE_ACCOUNT_EMAIL: "${SCHEDULER_SA}"
YAML

echo "==> Cloud Run サービスを更新する（${SERVICE}）"
# 無料枠に収める設定（COST）：使われないときは0台、最大2台、CPU はリクエストの処理中だけ割り当てる
gc run deploy "$SERVICE" --region "$REGION" --image "$image" \
  --service-account "$RUNTIME_SA" \
  --allow-unauthenticated \
  --min-instances 0 --max-instances 2 \
  --cpu 1 --memory 512Mi --cpu-throttling \
  --env-vars-file "$env_file" \
  --set-secrets "DATABASE_URL=${SECRET_DATABASE_URL}:latest,ADMIN_STUDENT_NUMBERS=${SECRET_ADMIN_STUDENT_NUMBERS}:latest,SMTP_PASSWORD=${SECRET_SMTP_PASSWORD}:latest"

echo "==> 稼働確認"
curl -fsS "${url}/api/health"
echo
echo "完了：${url}"
