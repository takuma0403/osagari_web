// Package storage はファイル保存（Cloud Storage）を抽象化する（STORAGE）。
// 画像はブラウザから署名付きURLで直接アップロードし、API はURLの発行とアップロード後の確認だけを行う（ARCH）。
// 本番は Cloud Storage、ローカルは fake-gcs-server（GCS）、テストは Memory を差し込む。
package storage

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Storage は署名付きURLの発行とオブジェクトの確認を行う。service はこのインターフェース越しに使う。
type Storage interface {
	// SignedPutURL は object に contentType の画像を PUT するための、expiresAt まで有効な署名付きURLを発行する。
	// 新しいオブジェクトを作るときだけ使え（上書き不可）、maxBytes を超える大きさは受け付けない。
	SignedPutURL(ctx context.Context, bucket, object, contentType string, maxBytes int64, expiresAt time.Time) (SignedURL, error)
	// SignedGetURL は object を expiresAt まで読める署名付きURLを発行する（学生証は管理者APIだけが使う。ADR 0015）。
	SignedGetURL(ctx context.Context, bucket, object string, expiresAt time.Time) (SignedURL, error)
	// Stat はオブジェクトの大きさと形式を返す。なければ ErrNotFound。
	Stat(ctx context.Context, bucket, object string) (ObjectInfo, error)
	// Delete はオブジェクトを消す。なければ ErrNotFound（学生証画像の削除。JOBS-1, ADR 0033）。
	Delete(ctx context.Context, bucket, object string) error
	// List はバケットのすべてのオブジェクトの名前と作成日時を返す（登録されなかった学生証の削除。ADR 0033）。
	List(ctx context.Context, bucket string) ([]ObjectEntry, error)
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

// putHeaders は署名に含める PUT のヘッダー。Content-Type 以外は Cloud Storage の XML API の条件：
// x-goog-if-generation-match: 0 でオブジェクトがまだないときだけ作れるようにし（登録した後に同じURLで画像を差し替えさせない）、
// x-goog-content-length-range で大きさの上限を付ける。
func putHeaders(contentType string, maxBytes int64) map[string]string {
	return map[string]string{
		"Content-Type":                contentType,
		"x-goog-if-generation-match":  "0",
		"x-goog-content-length-range": fmt.Sprintf("0,%d", maxBytes),
	}
}

func putURL(url string, headers map[string]string, expiresAt time.Time) SignedURL {
	return SignedURL{URL: url, Method: "PUT", Headers: headers, ExpiresAt: expiresAt}
}

func getURL(url string, expiresAt time.Time) SignedURL {
	return SignedURL{URL: url, Method: "GET", Headers: map[string]string{}, ExpiresAt: expiresAt}
}
