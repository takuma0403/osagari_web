#!/bin/sh
# S3 互換ストレージ（RustFS）に商品写真のバケットを作り、CORS を設定する（#141, STORAGE）。
# 何度実行しても同じ結果になる。CORS は本番の R2（infra/setup.sh）と同じく PUT だけを許可する。
set -eu

for i in $(seq 1 30); do
  aws --endpoint-url "$S3_URL" s3api list-buckets >/dev/null 2>&1 && break
  sleep 1
done

for b in $BUCKETS; do
  if aws --endpoint-url "$S3_URL" s3api head-bucket --bucket "$b" >/dev/null 2>&1; then
    echo "bucket $b: exists"
  else
    aws --endpoint-url "$S3_URL" s3api create-bucket --bucket "$b" >/dev/null
    echo "bucket $b: created"
  fi
  aws --endpoint-url "$S3_URL" s3api put-bucket-cors --bucket "$b" --cors-configuration "{
    \"CORSRules\": [{
      \"AllowedOrigins\": [\"$CORS_ORIGIN\"],
      \"AllowedMethods\": [\"PUT\"],
      \"AllowedHeaders\": [\"Content-Type\", \"If-None-Match\"],
      \"MaxAgeSeconds\": 3600
    }]
  }"
done
