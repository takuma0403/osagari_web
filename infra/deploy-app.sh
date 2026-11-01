#!/usr/bin/env bash
# 画面（app/）を Firebase Hosting に出す（#39, ADR 0045）：ビルド → デプロイ → 稼働確認。
# /api/** は firebase.json の rewrites で Cloud Run に転送するので、API の更新（deploy-server.sh）を先に済ませておく。
# 前提：Firebase がプロジェクトに追加済みであること（infra/README.md）。
set -euo pipefail
cd "$(dirname "$0")"
source ./config.sh

repo_root=$(git rev-parse --show-toplevel)
if [ -n "$(git status --porcelain -- "$repo_root/app" "$repo_root/firebase.json")" ]; then
  # どのコミットの画面が出ているかを特定できるよう、コミット済みのものだけを出す
  echo "app/ か firebase.json に未コミットの変更があります。コミットしてから実行してください" >&2
  exit 1
fi

echo "==> ビルドする"
(cd "$repo_root/app" && npm ci && npm run build)

echo "==> Firebase Hosting にデプロイする（${HOSTING_SITE}）"
(cd "$repo_root" && npx --yes "firebase-tools@${FIREBASE_TOOLS_VERSION}" deploy --only hosting \
  --project "$PROJECT_ID" --message "$(git rev-parse --short=12 HEAD)" --non-interactive)

echo "==> 稼働確認"
url="https://${HOSTING_SITE}.web.app"
curl -fsS "${url}/api/health"
echo
echo "完了：${url}"
