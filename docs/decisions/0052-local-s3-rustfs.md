# ADR 0052: ローカルの商品写真のストレージを RustFS にする

- 日付: 2026-11-19
- 状態: 採用
- 関連する章ID: STORAGE

## 背景

[ADR 0051](0051-item-images-on-cloudflare-r2.md) で、ローカルでは商品写真に S3 互換の MinIO を使うと決めた。#141 で compose に足そうとしたところ、MinIO の公式の Docker イメージ（Docker Hub の `minio/minio`、`quay.io/minio/minio`）が取得できなくなっていた。

ローカルのストレージには、本番の R2 と同じく次のことが求められる（[STORAGE](../tech/storage.md)）。

- 署名付き PUT URL の署名を検証し、署名に含めた `Content-Type`・`Content-Length`・`If-None-Match: *` と違うリクエストを拒む
- 同じオブジェクトへの2回目の PUT を `412` で拒む
- バケットの CORS 設定どおりに、ブラウザからの PUT のプリフライトを許可・拒否する

## 決定

- ローカルの商品写真のストレージは RustFS（`rustfs/rustfs`、Apache-2.0）にする。版はタグで固定する。
- バケットの作成と CORS の設定は、compose の初期化用コンテナ（`amazon/aws-cli`）で行う。CORS は本番の R2 と同じく PUT だけを許可し、オリジンは app の開発サーバー（`http://localhost:5173`）にする。
- 学生証は今までどおり fake-gcs-server。

## 理由と検討した選択肢

RustFS と versitygw の両方で、上の3つの条件を確かめた（違う `Content-Type`・`Content-Length`、`If-None-Match` なしは `403`、正しい PUT は `200`、2回目は `412`、許可していないオリジンのプリフライトは `403`）。

| 選択肢 | 良い点 | 悪い点 |
| --- | --- | --- |
| RustFS（採用） | MinIO の後継として作られた S3 互換サーバーで、使い方が近い。管理画面で中身を見られる | 1.0 が出てから日が浅い |
| versitygw | 小さく、版のリリースが安定している | 管理画面がない |
| MinIO の古いイメージを自前で用意する | ADR 0051 のまま | 公式に取得できず、更新もされない |
| 署名を検証しないモック（LocalStack・S3Mock など） | 手軽 | 本番の R2 と違い、署名の条件の誤りをローカルで見つけられない |

## 影響

- 更新したドキュメント：[tech/storage.md](../tech/storage.md)（STORAGE）、[ADR 0051](0051-item-images-on-cloudflare-r2.md)（置き換えた箇所の注記）
- 実装への影響：`server/compose.yaml`（`s3`・`s3-init`）、`server/dev/s3-init.sh`、`server/.env.example`（#141）
