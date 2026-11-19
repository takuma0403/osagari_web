// Package storage はファイル保存を抽象化する（STORAGE）。
// 画像はブラウザから署名付きURLで直接アップロードし、API はURLの発行とアップロード後の確認だけを行う（ARCH）。
// 学生証は Cloud Storage（GCS。ローカルは fake-gcs-server）、商品写真は Cloudflare R2（S3。ローカルは RustFS）に置き（ADR 0051）、
// テストは Memory を差し込む。
package storage

import (
	"context"
	"errors"
	"time"
)

// Storage は署名付きURLの発行とオブジェクトの確認を行う。service はこのインターフェース越しに使う。
type Storage interface {
	// SignedPutURL は object に opts.ContentType の画像を PUT するための、expiresAt まで有効な署名付きURLを発行する。
	// 新しいオブジェクトを作るときだけ使え（上書き不可）、opts.MaxBytes を超える大きさは受け付けない。
	SignedPutURL(ctx context.Context, bucket, object string, opts PutOptions, expiresAt time.Time) (SignedURL, error)
	// SignedGetURL は object を expiresAt まで読める署名付きURLを発行する（学生証は管理者APIだけが使う。ADR 0015）。
	SignedGetURL(ctx context.Context, bucket, object string, expiresAt time.Time) (SignedURL, error)
	// Stat はオブジェクトの大きさと形式を返す。なければ ErrNotFound。
	Stat(ctx context.Context, bucket, object string) (ObjectInfo, error)
	// Delete はオブジェクトを消す。なければ ErrNotFound（学生証画像の削除。JOBS-1, ADR 0033）。
	Delete(ctx context.Context, bucket, object string) error
	// List はバケットのすべてのオブジェクトの名前と作成日時を返す（登録されなかった学生証の削除。ADR 0033）。
	List(ctx context.Context, bucket string) ([]ObjectEntry, error)
}

// PutOptions は署名付き PUT URL の条件。
type PutOptions struct {
	// ContentType は画像の形式。署名に含めるので、ほかの形式では PUT できない
	ContentType string
	// MaxBytes は大きさの上限
	MaxBytes int64
	// Size はブラウザが送る画像の大きさ（バイト）。S3 は大きさの範囲の条件を持たないので、この値を Content-Length として署名に含める（必須。ADR 0051）。
	// GCS は MaxBytes の範囲で守るので使わない（0 でよい）
	Size int64
}

// SignedURL はブラウザに渡す署名付きURL。Headers は署名に含まれるので、リクエストにそのまま付ける（GET では空）。
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

// ObjectEntry はバケットの一覧の1件。
type ObjectEntry struct {
	Name    string
	Created time.Time
}

// ErrNotFound はオブジェクトがないときに返す。
var ErrNotFound = errors.New("storage: オブジェクトがありません")

func putURL(url string, headers map[string]string, expiresAt time.Time) SignedURL {
	return SignedURL{URL: url, Method: "PUT", Headers: headers, ExpiresAt: expiresAt}
}

func getURL(url string, expiresAt time.Time) SignedURL {
	return SignedURL{URL: url, Method: "GET", Headers: map[string]string{}, ExpiresAt: expiresAt}
}
