package storage

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

// S3Config は S3 互換ストレージ（本番は Cloudflare R2、ローカルは RustFS）の設定（STORAGE, ADR 0051・0052）。
type S3Config struct {
	// Endpoint は API からの接続先（例 https://<アカウントID>.r2.cloudflarestorage.com、ローカルは http://s3:9000）
	Endpoint string
	// PublicEndpoint は署名付きURLに入れる、ブラウザから届く接続先（ローカルは http://localhost:9000）。空なら Endpoint
	PublicEndpoint string
	// Region は署名に使うリージョン（R2 は auto）
	Region string
	// AccessKeyID・SecretAccessKey は署名に使うアクセスキー（本番は Secret Manager の R2 API トークン）
	AccessKeyID     string
	SecretAccessKey string
}

// S3 は S3 互換ストレージの実装。
type S3 struct {
	// client は API からの操作（Stat・Delete・List）に使う
	client *s3.Client
	// presign はブラウザに渡す署名付きURLの発行に使う（ホストを PublicEndpoint にする）
	presign *s3.PresignClient
}

var _ Storage = (*S3)(nil)

// NewS3 は S3 互換ストレージのクライアントを作る。
func NewS3(cfg S3Config) (*S3, error) {
	if cfg.Endpoint == "" || cfg.Region == "" || cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return nil, errors.New("storage: S3 の接続先・リージョン・アクセスキーが足りません")
	}
	public := cfg.PublicEndpoint
	if public == "" {
		public = cfg.Endpoint
	}
	newClient := func(endpoint string) *s3.Client {
		return s3.New(s3.Options{
			BaseEndpoint: aws.String(endpoint),
			Region:       cfg.Region,
			Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
			// バケット名をホストに入れない（RustFS はパス形式だけ。R2 はどちらも受け付ける）
			UsePathStyle: true,
			// SDK が既定で付けるチェックサムを付けない。署名付きURLに入るとブラウザの PUT が合わなくなり、R2 も一部に対応していない
			RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
			ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
		})
	}
	return &S3{
		client:  newClient(cfg.Endpoint),
		presign: s3.NewPresignClient(newClient(public)),
	}, nil
}

// SignedPutURL は SigV4 の PUT 用URLを発行する。Content-Type・Content-Length（opts.Size）・If-None-Match: * を署名に含めるので、
// ほかの形式・ほかの大きさ・上書きでは PUT できない（ADR 0051）。
// 返すヘッダーは Content-Type と If-None-Match だけ。Content-Length はブラウザが本文から付ける（JS からは設定できない）。
func (c *S3) SignedPutURL(ctx context.Context, bucket, object string, opts PutOptions, expiresAt time.Time) (SignedURL, error) {
	if opts.Size <= 0 || opts.Size > opts.MaxBytes {
		return SignedURL{}, fmt.Errorf("storage: 画像の大きさ %d が範囲外です（上限 %d）", opts.Size, opts.MaxBytes)
	}
	req, err := c.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(object),
		ContentType:   aws.String(opts.ContentType),
		ContentLength: aws.Int64(opts.Size),
		IfNoneMatch:   aws.String("*"),
	}, s3.WithPresignExpires(time.Until(expiresAt)))
	if err != nil {
		return SignedURL{}, fmt.Errorf("storage: 署名付きURLを発行できません: %w", err)
	}
	headers := map[string]string{
		"Content-Type":  opts.ContentType,
		"If-None-Match": "*",
	}
	return putURL(req.URL, headers, expiresAt), nil
}

// SignedGetURL は SigV4 の GET 用URLを発行する。
func (c *S3) SignedGetURL(ctx context.Context, bucket, object string, expiresAt time.Time) (SignedURL, error) {
	req, err := c.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(object),
	}, s3.WithPresignExpires(time.Until(expiresAt)))
	if err != nil {
		return SignedURL{}, fmt.Errorf("storage: 署名付きURLを発行できません: %w", err)
	}
	return getURL(req.URL, expiresAt), nil
}

// Stat はオブジェクトの属性を読む。
func (c *S3) Stat(ctx context.Context, bucket, object string) (ObjectInfo, error) {
	out, err := c.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(object),
	})
	if isS3NotFound(err) {
		return ObjectInfo{}, ErrNotFound
	}
	if err != nil {
		return ObjectInfo{}, fmt.Errorf("storage: オブジェクトを確認できません: %w", err)
	}
	return ObjectInfo{Size: aws.ToInt64(out.ContentLength), ContentType: aws.ToString(out.ContentType)}, nil
}

// Delete はオブジェクトを消す。S3 の DeleteObject はオブジェクトがなくても成功するので、先に有無を確かめて ErrNotFound を返す。
func (c *S3) Delete(ctx context.Context, bucket, object string) error {
	if _, err := c.Stat(ctx, bucket, object); err != nil {
		return err
	}
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(object),
	})
	if err != nil {
		return fmt.Errorf("storage: オブジェクトを消せません: %w", err)
	}
	return nil
}

// List はバケットのオブジェクトを名前と作成日時（S3 では最終更新日時。上書きしないので作成日時と同じ）だけ読む。
func (c *S3) List(ctx context.Context, bucket string) ([]ObjectEntry, error) {
	var out []ObjectEntry
	p := s3.NewListObjectsV2Paginator(c.client, &s3.ListObjectsV2Input{Bucket: aws.String(bucket)})
	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("storage: オブジェクトの一覧を読めません: %w", err)
		}
		for _, o := range page.Contents {
			out = append(out, ObjectEntry{Name: aws.ToString(o.Key), Created: aws.ToTime(o.LastModified)})
		}
	}
	return out, nil
}

// isS3NotFound はオブジェクトがないことを表すエラーかを返す。HeadObject は本文がないので、型か HTTP の 404 で判定する。
func isS3NotFound(err error) bool {
	if err == nil {
		return false
	}
	var nf *types.NotFound
	var nsk *types.NoSuchKey
	if errors.As(err, &nf) || errors.As(err, &nsk) {
		return true
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) && (apiErr.ErrorCode() == "NotFound" || apiErr.ErrorCode() == "NoSuchKey") {
		return true
	}
	var re interface{ HTTPStatusCode() int }
	return errors.As(err, &re) && re.HTTPStatusCode() == http.StatusNotFound
}
