// Package storage はファイル保存（Cloud Storage）を抽象化する（STORAGE）。
// 画像はブラウザから署名付きURLで直接アップロードし、API はURLの発行とアップロード後の確認だけを行う（ARCH）。
// 本番は Cloud Storage、ローカルは fake-gcs-server（GCS）、テストは Memory を差し込む。
package storage

import (
	"context"
	"errors"
	"time"
)

// Storage は署名付きURLの発行とオブジェクトの確認を行う。service はこのインターフェース越しに使う。
type Storage interface {
	// SignedPutURL は object に contentType の画像を PUT するための、expiresAt まで有効な署名付きURLを発行する。
	SignedPutURL(ctx context.Context, bucket, object, contentType string, expiresAt time.Time) (SignedURL, error)
	// Stat はオブジェクトの大きさと形式を返す。なければ ErrNotFound。
	Stat(ctx context.Context, bucket, object string) (ObjectInfo, error)
}

// SignedURL はブラウザに渡すアップロード先。Headers は署名に含まれるので、PUT にそのまま付ける。
type SignedURL struct {
	URL       string
	Method    string
	Headers   map[string]string
	ExpiresAt time.Time
}

// ObjectInfo はアップロードされたオブジェクトの情報。
type ObjectInfo struct {
	Size        int64
	ContentType string
}

// ErrNotFound はオブジェクトがないときに返す。
var ErrNotFound = errors.New("storage: オブジェクトがありません")

func putURL(url, contentType string, expiresAt time.Time) SignedURL {
	return SignedURL{
		URL:       url,
		Method:    "PUT",
		Headers:   map[string]string{"Content-Type": contentType},
		ExpiresAt: expiresAt,
	}
}
