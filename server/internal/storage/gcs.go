package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	gcs "cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
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

// SignedPutURL は V4 署名の PUT 用URLを発行する。Content-Type と条件のヘッダー（putHeaders）は署名に含めるので、
// ほかの形式・上書き・上限を超える大きさでは PUT できない。
func (g *GCS) SignedPutURL(_ context.Context, bucket, object, contentType string, maxBytes int64, expiresAt time.Time) (SignedURL, error) {
	headers := putHeaders(contentType, maxBytes)
	opts := &gcs.SignedURLOptions{
		Scheme:      gcs.SigningSchemeV4,
		Method:      "PUT",
		Expires:     expiresAt,
		ContentType: contentType,
	}
	for k, v := range headers {
		if k != "Content-Type" {
			opts.Headers = append(opts.Headers, k+":"+v)
		}
	}
	url, err := g.signedURL(bucket, object, opts)
	if err != nil {
		return SignedURL{}, err
	}
	return putURL(url, headers, expiresAt), nil
}

// SignedGetURL は V4 署名の GET 用URLを発行する。
func (g *GCS) SignedGetURL(_ context.Context, bucket, object string, expiresAt time.Time) (SignedURL, error) {
	url, err := g.signedURL(bucket, object, &gcs.SignedURLOptions{
		Scheme:  gcs.SigningSchemeV4,
		Method:  "GET",
		Expires: expiresAt,
	})
	if err != nil {
		return SignedURL{}, err
	}
	return getURL(url, expiresAt), nil
}

// signedURL は opts で署名する。本番は鍵ファイルを持たず、SDK がサービスアカウントの IAM signBlob で署名する。
func (g *GCS) signedURL(bucket, object string, opts *gcs.SignedURLOptions) (string, error) {
	if g.cfg.EmulatorPublicHost != "" {
		// fake-gcs-server は署名を検証しないので、ダミーの署名で URL の形だけを作る
		opts.GoogleAccessID = "local@osagari.local"
		opts.SignBytes = func([]byte) ([]byte, error) { return []byte("local"), nil }
		opts.Hostname = g.cfg.EmulatorPublicHost
		opts.Insecure = true
	}
	url, err := g.client.Bucket(bucket).SignedURL(object, opts)
	if err != nil {
		return "", fmt.Errorf("storage: 署名付きURLを発行できません: %w", err)
	}
	return url, nil
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

// Delete はオブジェクトを消す。
func (g *GCS) Delete(ctx context.Context, bucket, object string) error {
	err := g.client.Bucket(bucket).Object(object).Delete(ctx)
	if errors.Is(err, gcs.ErrObjectNotExist) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("storage: オブジェクトを消せません: %w", err)
	}
	return nil
}

// List はバケットのオブジェクトを名前と作成日時だけ読む。
func (g *GCS) List(ctx context.Context, bucket string) ([]ObjectEntry, error) {
	q := &gcs.Query{}
	if err := q.SetAttrSelection([]string{"Name", "Created"}); err != nil {
		return nil, fmt.Errorf("storage: 一覧の条件を作れません: %w", err)
	}
	var out []ObjectEntry
	it := g.client.Bucket(bucket).Objects(ctx, q)
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			return out, nil
		}
		if err != nil {
			return nil, fmt.Errorf("storage: オブジェクトの一覧を読めません: %w", err)
		}
		out = append(out, ObjectEntry{Name: attrs.Name, Created: attrs.Created})
	}
}
