#!/bin/sh
# fake-gcs-server にバケットを作る。既にあれば何もしない。
set -eu

for i in $(seq 1 30); do
  curl -sf "$STORAGE_URL/storage/v1/b" >/dev/null && break
  sleep 1
done

for b in $BUCKETS; do
  if curl -sf "$STORAGE_URL/storage/v1/b/$b" >/dev/null; then
    echo "bucket $b: exists"
  else
    curl -sf -X POST -H 'Content-Type: application/json' \
      -d "{\"name\":\"$b\"}" "$STORAGE_URL/storage/v1/b?project=local" >/dev/null
    echo "bucket $b: created"
  fi
done
