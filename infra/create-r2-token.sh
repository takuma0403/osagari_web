#!/usr/bin/env bash
# 商品写真の R2 バケットだけを読み書きできる R2 API トークンを作り、S3 互換 API のアクセスキーを Secret Manager に入れる（#141, STORAGE, ADR 0051）。
# 値は画面にもログにも出さない。入れ替えにも使う（新しいトークンを作る → deploy-server.sh → prune で古いトークンを消す）。
#
#   infra/create-r2-token.sh          新しいトークンを作り、アクセスキーを Secret Manager に入れる
#   infra/create-r2-token.sh prune    Secret Manager のアクセスキーと違う、このスクリプトが作ったトークンを消す
#
# Cloudflare API を呼ぶために「アカウント API トークンの編集」権限を持つトークン（作業用）が要る。環境変数 CLOUDFLARE_API_TOKEN か、
# 入力で渡す（入力は画面に表示されない）。作業用のトークンはダッシュボードで作り、終わったら消す（infra/README.md）。
set -euo pipefail
cd "$(dirname "$0")"
source ./config.sh
require_r2_account_id

mode=${1:-create}
# このスクリプトが作るトークンの名前の接頭辞。prune はこの名前のトークンだけを消す
token_prefix="osagari-api ${R2_BUCKET_ITEM_IMAGES}"
api="https://api.cloudflare.com/client/v4/accounts/${R2_ACCOUNT_ID}/tokens"

if [ -z "${CLOUDFLARE_API_TOKEN:-}" ]; then
  read -rsp "作業用の Cloudflare API トークン: " CLOUDFLARE_API_TOKEN
  echo
fi

# 認証ヘッダーはコマンドライン引数に載せない（プロセス一覧から見えないように）
cf() {
  local method=$1 path=$2
  shift 2
  local res
  res=$(curl -sS -X "$method" "${api}${path}" \
    -H @<(printf 'Authorization: Bearer %s\n' "$CLOUDFLARE_API_TOKEN") \
    -H 'Content-Type: application/json' "$@")
  if [ "$(jq -r '.success' <<<"$res")" != "true" ]; then
    # エラーには値が入らないので、内容をそのまま出す
    echo "Cloudflare API がエラーを返しました（${method} ${path}）：$(jq -c '.errors' <<<"$res")" >&2
    exit 1
  fi
  printf '%s' "$res"
}

case "$mode" in
create)
  echo "==> R2 の権限グループを調べる"
  groups=$(cf GET /permission_groups)
  read_id=$(jq -r '.result[] | select(.name == "Workers R2 Storage Bucket Item Read") | .id' <<<"$groups")
  write_id=$(jq -r '.result[] | select(.name == "Workers R2 Storage Bucket Item Write") | .id' <<<"$groups")
  if [ -z "$read_id" ] || [ -z "$write_id" ]; then
    echo "R2 のオブジェクトの読み書きの権限グループが見つかりません" >&2
    exit 1
  fi

  echo "==> トークンを作る（${R2_BUCKET_ITEM_IMAGES} のオブジェクトの読み書きだけ）"
  body=$(jq -n --arg name "${token_prefix} $(date +%Y%m%d-%H%M%S)" \
    --arg resource "com.cloudflare.edge.r2.bucket.${R2_ACCOUNT_ID}_default_${R2_BUCKET_ITEM_IMAGES}" \
    --arg read "$read_id" --arg write "$write_id" \
    '{name: $name, policies: [{effect: "allow", resources: {($resource): "*"}, permission_groups: [{id: $read}, {id: $write}]}]}')
  created=$(cf POST "" --data @<(printf '%s' "$body"))

  # S3 互換 API のアクセスキーID はトークンの ID、シークレットアクセスキーはトークンの値の SHA-256（R2 の仕様）
  jq -j '.result.id' <<<"$created" | ./set-secret.sh "$SECRET_R2_ACCESS_KEY_ID"
  jq -j '.result.value' <<<"$created" | sha256sum | cut -d' ' -f1 | tr -d '\n' | ./set-secret.sh "$SECRET_R2_SECRET_ACCESS_KEY"
  echo "完了。infra/deploy-server.sh で反映したあと、古いトークンがあれば infra/create-r2-token.sh prune で消す"
  ;;
prune)
  current=$(gcloud --project "$PROJECT_ID" --quiet secrets versions access latest --secret "$SECRET_R2_ACCESS_KEY_ID")
  tokens=$(cf GET "?per_page=50")
  old=$(jq -r --arg prefix "$token_prefix" --arg current "$current" \
    '.result[] | select((.name | startswith($prefix)) and .id != $current) | .id' <<<"$tokens")
  if [ -z "$old" ]; then
    echo "消すトークンはありません"
    exit 0
  fi
  for id in $old; do
    name=$(jq -r --arg id "$id" '.result[] | select(.id == $id) | .name' <<<"$tokens")
    cf DELETE "/${id}" >/dev/null
    echo "==> ${name} を消しました"
  done
  ;;
*)
  echo "使い方: create-r2-token.sh [create|prune]" >&2
  exit 1
  ;;
esac
