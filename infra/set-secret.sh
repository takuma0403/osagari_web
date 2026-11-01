#!/usr/bin/env bash
# 標準入力の値をシークレットの新しいバージョンとして追加し、古いバージョンを破棄する（#38, ADR 0042）。
# 値は画面にもログにも出さない。Secret Manager の無料枠（有効なバージョン6個）に収めるため、有効なバージョンは1つだけにする。
#
#   printf '%s' "$値" | infra/set-secret.sh database-url
set -euo pipefail
cd "$(dirname "$0")"
source ./config.sh

name=${1:?使い方: set-secret.sh <シークレット名>（値は標準入力）}
gc() { gcloud --project "$PROJECT_ID" --quiet "$@"; }

new=$(gc secrets versions add "$name" --data-file=- --format 'value(name)')
new=${new##*/}
echo "==> ${name} のバージョン ${new} を追加しました"

for v in $(gc secrets versions list "$name" --filter 'state!=DESTROYED' --format 'value(name)'); do
  v=${v##*/}
  if [ "$v" != "$new" ]; then
    gc secrets versions destroy "$v" --secret "$name" >/dev/null
    echo "==> ${name} の古いバージョン ${v} を破棄しました"
  fi
done
echo "反映するには infra/deploy-server.sh を実行する（Cloud Run は起動時に最新版を読む）"
