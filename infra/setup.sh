#!/usr/bin/env bash
# 本番環境の土台を作る（#38, ADR 0042）。何度実行しても同じ結果になる。
# 前提：GCP プロジェクトが作成済みで、請求先アカウントに紐づいていること（手順は infra/README.md）。
set -euo pipefail
cd "$(dirname "$0")"
source ./config.sh

gc() { gcloud --project "$PROJECT_ID" --quiet "$@"; }

echo "==> API を有効にする"
gc services enable run.googleapis.com artifactregistry.googleapis.com secretmanager.googleapis.com iam.googleapis.com

echo "==> Artifact Registry のリポジトリ"
if ! gc artifacts repositories describe "$AR_REPO" --location "$REGION" >/dev/null 2>&1; then
  gc artifacts repositories create "$AR_REPO" --location "$REGION" --repository-format docker \
    --description "osagari の API イメージ"
fi
# 無料枠（0.5GB）に収めるため、最新3件より古いイメージを消す（COST）
policy=$(mktemp)
trap 'rm -f "$policy"' EXIT
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

echo "完了。次はシークレットの値を入れる（infra/README.md）"
