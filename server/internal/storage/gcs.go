package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	gcs "cloud.google.com/go/storage"
)

// GCSConfig は Cloud Storage の設定。
type GCSConfig struct {
	// EmulatorPublicHost はエミュレータ（fake-gcs-server）を使うときに、署名付きURLに入れるブラウザから届くホスト（例 localhost:4443）。
	// 空なら本番の Cloud Storage として署名する。エミュレータへの接続先は SDK が STORAGE_EMULATOR_HOST から読む
	EmulatorPublicHost string
}

// GCS は Cloud Storage の実装。
type GCS struct {
	client *gcs.Client
	cfg    GCSConfig
}

var _ Storage = (*GCS)(nil)

// NewGCS は Cloud Storage のクライアントを作る。本番は Cloud Run のサービスアカウントの認証情報を使う。
func NewGCS(ctx context.Context, cfg GCSConfig) (*GCS, error) {
	client, err := gcs.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage: クライアントを作れません: %w", err)
	}
	return &GCS{client: client, cfg: cfg}, nil
}

// Close はクライアントを閉じる。
func (g *GCS) Close() error { return g.client.Close() }

// SignedPutURL は V4 署名の PUT 用URLを発行する。Content-Type は署名に含めるので、ほかの形式では PUT できない。
// 本番は鍵ファイルを持たず、SDK がサービスアカウントの IAM signBlob で署名する。
func (g *GCS) SignedPutURL(_ context.Context, bucket, object, contentType string, expiresAt time.Time) (SignedURL, error) {
	opts := &gcs.SignedURLOptions{
		Scheme:      gcs.SigningSchemeV4,
		Method:      "PUT",
		Expires:     expiresAt,
		ContentType: contentType,
	}
	if g.cfg.EmulatorPublicHost != "" {
		// fake-gcs-server は署名を検証しないので、ダミーの署名で URL の形だけを作る
		opts.GoogleAccessID = "local@osagari.local"
		opts.SignBytes = func([]byte) ([]byte, error) { return []byte("local"), nil }
		opts.Hostname = g.cfg.EmulatorPublicHost
		opts.Insecure = true
	}
	url, err := g.client.Bucket(bucket).SignedURL(object, opts)
	if err != nil {
		return SignedURL{}, fmt.Errorf("storage: 署名付きURLを発行できません: %w", err)
	}
	return putURL(url, contentType, expiresAt), nil
}

// Stat はオブジェクトの属性を読む。
func (g *GCS) Stat(ctx context.Context, bucket, object string) (ObjectInfo, error) {
	attrs, err := g.client.Bucket(bucket).Object(object).Attrs(ctx)
	if errors.Is(err, gcs.ErrObjectNotExist) {
		return ObjectInfo{}, ErrNotFound
	}
	if err != nil {
		return ObjectInfo{}, fmt.Errorf("storage: オブジェクトを確認できません: %w", err)
	}
	return ObjectInfo{Size: attrs.Size, ContentType: attrs.ContentType}, nil
}
