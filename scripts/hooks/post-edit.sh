#!/usr/bin/env bash
# PostToolUse フック（Edit / Write / MultiEdit の後に実行）
# - docs や指示ファイル（*.md / openapi.yaml）を変更 → check-docs.mjs を実行し、失敗したら exit 2 で Claude に知らせる
# - *.go を変更 → gofmt
# - app/ の TS・CSS を変更 → app/node_modules があるときだけ prettier
set -uo pipefail

ROOT="${CLAUDE_PROJECT_DIR:-$(cd "$(dirname "$0")/../.." && pwd)}"
INPUT="$(cat)"
FILE="$(printf '%s' "$INPUT" | node -e '
  let s = ""; process.stdin.on("data", (d) => (s += d)).on("end", () => {
    try { const j = JSON.parse(s); process.stdout.write(j.tool_input?.file_path ?? j.tool_input?.path ?? ""); } catch {}
  });')"

[ -z "$FILE" ] && exit 0
REL="${FILE#"$ROOT"/}"

case "$REL" in
  docs/_source/*)
    echo "docs/_source/ は分割前の凍結コピーです。書き換えず、docs/product・docs/tech を編集してください。" >&2
    exit 2
    ;;
  docs/*|CLAUDE.md|AGENTS.md|README.md|app/CLAUDE.md|server/CLAUDE.md|.claude/*.md|.claude/*/*.md|.claude/*/*/*.md)
    if ! OUT="$(node "$ROOT/scripts/check-docs.mjs" 2>&1)"; then
      echo "$OUT" >&2
      exit 2
    fi
    ;;
  *.go)
    command -v gofmt >/dev/null && gofmt -w "$FILE"
    ;;
  app/*.ts|app/*.tsx|app/*.css|app/*.json)
    if [ -x "$ROOT/app/node_modules/.bin/prettier" ]; then
      "$ROOT/app/node_modules/.bin/prettier" --write --log-level warn "$FILE" || true
    fi
    ;;
esac
exit 0
