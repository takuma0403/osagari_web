#!/usr/bin/env bash
# 本番環境の土台を作る（#38, ADR 0042）。何度実行しても同じ結果になる。
# 前提：GCP プロジェクトが作成済みで、請求先アカウントに紐づいていること（手順は infra/README.md）。
# 初回は通知先のメールアドレスを ALERT_EMAIL で渡す（通知チャンネルを作ったあとは不要）。
set -euo pipefail
cd "$(dirname "$0")"
source ./config.sh

gc() { gcloud --project "$PROJECT_ID" --quiet "$@"; }

echo "==> API を有効にする"
# iamcredentials は署名付きURLの署名（IAM signBlob）と GitHub Actions からの SA の借用に使う（STORAGE, #39）。
# logging・monitoring は送信数の通知に使う（AUTHN-6）。sts は Workload Identity 連携、firebasehosting は画面の配信に使う（#39）。
# cloudscheduler は定期処理（JOBS-1）に使う（#40）
gc services enable run.googleapis.com artifactregistry.googleapis.com secretmanager.googleapis.com iam.googleapis.com \
  storage.googleapis.com iamcredentials.googleapis.com logging.googleapis.com monitoring.googleapis.com \
  sts.googleapis.com firebasehosting.googleapis.com cloudscheduler.googleapis.com

echo "==> Artifact Registry のリポジトリ"
if ! gc artifacts repositories describe "$AR_REPO" --location "$REGION" >/dev/null 2>&1; then
  gc artifacts repositories create "$AR_REPO" --location "$REGION" --repository-format docker \
    --description "osagari の API イメージ"
fi
# 無料枠（0.5GB）に収めるため、最新3件より古いイメージを消す（COST）
policy=$(mktemp)
cors=$(mktemp)
alert=$(mktemp)
trap 'rm -f "$policy" "$cors" "$alert"' EXIT
cat >"$policy" <<'JSON'
[
  {"name": "keep-latest-3", "action": {"type": "Keep"}, "mostRecentVersions": {"keepCount": 3}},
  {"name": "delete-others", "action": {"type": "Delete"}, "condition": {"tagState": "ANY"}}
]
JSON
gc artifacts repositories set-cleanup-policies "$AR_REPO" --location "$REGION" --policy "$policy" --no-dry-run >/dev/null

echo "==> 実行用のサービスアカウント"
if ! gc iam service-accounts describe "$RUNTIME_SA" >/dev/null 2>&1; then
  gc iam service-accounts create "${RUNTIME_SA%%@*}" --display-name "osagari API（Cloud Run）"
fi

echo "==> シークレットの器と読み取り権限"
for name in "${SECRETS[@]}"; do
  if ! gc secrets describe "$name" >/dev/null 2>&1; then
    gc secrets create "$name" --replication-policy automatic
  fi
  # 権限はシークレット単位で付ける（プロジェクト全体には付けない）
  gc secrets add-iam-policy-binding "$name" \
    --member "serviceAccount:${RUNTIME_SA}" --role roles/secretmanager.secretAccessor >/dev/null
done

echo "==> Cloud Storage のバケット（#69, STORAGE）"
# ブラウザが署名付きURLに直接 PUT するときのヘッダー（server/internal/storage の putHeaders と合わせる）
origins=$(printf '"%s",' "${HOSTING_ORIGINS[@]}")
cat >"$cors" <<JSON
[
  {
    "origin": [${origins%,}],
    "method": ["PUT"],
    "responseHeader": ["Content-Type", "x-goog-if-generation-match", "x-goog-content-length-range"],
    "maxAgeSeconds": 3600
  }
]
JSON
for bucket in "${BUCKETS[@]}"; do
  if ! gc storage buckets describe "gs://${bucket}" >/dev/null 2>&1; then
    gc storage buckets create "gs://${bucket}" --location "$STORAGE_LOCATION" \
      --default-storage-class STANDARD --uniform-bucket-level-access
  fi
  # 商品写真も学生証も署名付きURLでしか見せないので、どちらも公開アクセス防止を付ける。
  # ソフト削除は無効にする（期限が来た学生証を確実に消し、削除後に無料枠の容量を使わないため）
  gc storage buckets update "gs://${bucket}" --uniform-bucket-level-access --public-access-prevention \
    --clear-soft-delete --cors-file "$cors" >/dev/null
  # 権限はバケット単位で付ける（プロジェクト全体には付けない）
  gc storage buckets add-iam-policy-binding "gs://${bucket}" \
    --member "serviceAccount:${RUNTIME_SA}" --role roles/storage.objectAdmin >/dev/null
done

echo "==> 署名付きURLの署名の権限"
# 鍵ファイルを持たず、自分自身の IAM signBlob で署名するため（STORAGE, #15）
gc iam service-accounts add-iam-policy-binding "$RUNTIME_SA" \
  --member "serviceAccount:${RUNTIME_SA}" --role roles/iam.serviceAccountTokenCreator >/dev/null

echo "==> コードのメールの送信数の通知（#41, AUTHN-6）"
# 通知チャンネルは表示名で探し、なければ ALERT_EMAIL のアドレスで作る。通知先のアドレスはリポジトリに書かない
channel=$(gc beta monitoring channels list --filter "displayName=\"${ALERT_CHANNEL_NAME}\"" --format 'value(name)' | head -n1)
if [ -z "$channel" ]; then
  : "${ALERT_EMAIL:?通知チャンネルがまだないので、通知先のメールアドレスを ALERT_EMAIL で渡してください（infra/README.md）}"
  channel=$(gc beta monitoring channels create --display-name "$ALERT_CHANNEL_NAME" --type email \
    --channel-labels "email_address=${ALERT_EMAIL}" --format 'value(name)')
fi
# 直近24時間の送信が上限に近づくと API が WARN のログを出す（server/internal/service/code.go の warnIfManyMails）。
# ログが出るたびに通知されないよう、通知は1時間に1回までにする
if [ -z "$(gc monitoring policies list --filter "displayName=\"${ALERT_POLICY_NAME}\"" --format 'value(name)')" ]; then
  cat >"$alert" <<JSON
{
  "displayName": "${ALERT_POLICY_NAME}",
  "severity": "WARNING",
  "combiner": "OR",
  "conditions": [
    {
      "displayName": "送信数の警告のログが出た",
      "conditionMatchedLog": {
        "filter": "resource.type=\"cloud_run_revision\" AND resource.labels.service_name=\"${SERVICE}\" AND severity=WARNING AND jsonPayload.message=\"コードのメールの送信数が1日の上限に近づいています\""
      }
    }
  ],
  "alertStrategy": {
    "notificationRateLimit": {"period": "3600s"},
    "autoClose": "86400s"
  },
  "notificationChannels": ["${channel}"],
  "documentation": {
    "mimeType": "text/markdown",
    "content": "無料の Gmail は1日およそ500通までしか送れません。直近24時間に送ったコードのメールが400通に達しました（AUTHN-6）。送信は止めていません。ログで送信先の偏りや不正な利用がないかを確認してください。"
  }
}
JSON
  gc monitoring policies create --policy-from-file "$alert" >/dev/null
fi

echo "==> GitHub Actions からのデプロイ（#39, ADR 0045）"
if ! gc iam service-accounts describe "$DEPLOYER_SA" >/dev/null 2>&1; then
  gc iam service-accounts create "${DEPLOYER_SA%%@*}" --display-name "osagari デプロイ（GitHub Actions）"
fi
# deploy-server.sh：Cloud Run のサービスとジョブの更新（--allow-unauthenticated で IAM も設定するので admin）、
# プロジェクト番号の取得（browser）。deploy-app.sh：Hosting のデプロイと、API が有効かの確認（serviceUsageViewer）
for role in roles/run.admin roles/browser roles/firebasehosting.admin roles/serviceusage.serviceUsageViewer; do
  gc projects add-iam-policy-binding "$PROJECT_ID" \
    --member "serviceAccount:${DEPLOYER_SA}" --role "$role" --condition None >/dev/null
done
# イメージの push はリポジトリ単位、実行用 SA でのデプロイはその SA 単位で許可する（プロジェクト全体には付けない）
gc artifacts repositories add-iam-policy-binding "$AR_REPO" --location "$REGION" \
  --member "serviceAccount:${DEPLOYER_SA}" --role roles/artifactregistry.writer >/dev/null
gc iam service-accounts add-iam-policy-binding "$RUNTIME_SA" \
  --member "serviceAccount:${DEPLOYER_SA}" --role roles/iam.serviceAccountUser >/dev/null

if ! gc iam workload-identity-pools describe "$WIF_POOL" --location global >/dev/null 2>&1; then
  gc iam workload-identity-pools create "$WIF_POOL" --location global --display-name "GitHub Actions"
fi
# このリポジトリの main ブランチの workflow だけを受け入れる。dev やほかのブランチ・フォークからは本番に出せない
oidc_args=(
  --workload-identity-pool "$WIF_POOL" --location global
  --issuer-uri https://token.actions.githubusercontent.com
  --attribute-mapping "google.subject=assertion.sub,attribute.repository=assertion.repository,attribute.ref=assertion.ref"
  --attribute-condition "assertion.repository == '${GITHUB_REPO}' && assertion.ref == 'refs/heads/main'"
)
if gc iam workload-identity-pools providers describe "$WIF_PROVIDER" --workload-identity-pool "$WIF_POOL" --location global >/dev/null 2>&1; then
  gc iam workload-identity-pools providers update-oidc "$WIF_PROVIDER" "${oidc_args[@]}" >/dev/null
else
  gc iam workload-identity-pools providers create-oidc "$WIF_PROVIDER" "${oidc_args[@]}" --display-name "GitHub Actions"
fi
pool=$(gc iam workload-identity-pools describe "$WIF_POOL" --location global --format 'value(name)')
gc iam service-accounts add-iam-policy-binding "$DEPLOYER_SA" --role roles/iam.workloadIdentityUser \
  --member "principalSet://iam.googleapis.com/${pool}/attribute.repository/${GITHUB_REPO}" >/dev/null

echo "==> 定期処理の Cloud Scheduler（#40, JOBS-1）"
if ! gc iam service-accounts describe "$SCHEDULER_SA" >/dev/null 2>&1; then
  gc iam service-accounts create "${SCHEDULER_SA%%@*}" --display-name "osagari 定期処理（Cloud Scheduler）"
fi
# API は公開のサービスなので、この SA にロールは付けない。Cloud Scheduler が付ける OIDC ID トークンの
# audience とメールアドレスをアプリが照合する（ADR 0027）。audience は deploy-server.sh の JOBS_OIDC_AUDIENCE と同じ値
url=$(service_url)
# ジョブは失敗すると 500 を返すので、間をあけて再試行させる（JOBS-1）。期限は Cloud Run のリクエストの既定の期限に合わせる
scheduler_args=(
  --location "$SCHEDULER_LOCATION" --time-zone "$SCHEDULER_TIME_ZONE"
  --http-method POST --attempt-deadline 300s
  --oidc-service-account-email "$SCHEDULER_SA" --oidc-token-audience "$url"
  --max-retry-attempts 3 --min-backoff 60s
)
upsert_scheduler_job() {
  local name=$1 schedule=$2 path=$3 action=create
  if gc scheduler jobs describe "$name" --location "$SCHEDULER_LOCATION" >/dev/null 2>&1; then
    action=update
  fi
  gc scheduler jobs "$action" http "$name" "${scheduler_args[@]}" --schedule "$schedule" --uri "${url}${path}" >/dev/null
}
upsert_scheduler_job "$SCHEDULER_JOB_SHORT" "$SCHEDULER_SCHEDULE_SHORT" /api/internal/jobs/short
upsert_scheduler_job "$SCHEDULER_JOB_DAILY" "$SCHEDULER_SCHEDULE_DAILY" /api/internal/jobs/daily

echo "完了。次はシークレットの値を入れる（infra/README.md）"
