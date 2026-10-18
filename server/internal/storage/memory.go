package storage

import (
	"context"
	"net/url"
	"sync"
	"time"
)

// Memory はテスト用の Storage。オブジェクトをメモリに持ち、Put でアップロードしたことにする。並行に使ってよい。
type Memory struct {
	mu      sync.Mutex
	objects map[string]ObjectInfo
	// Err を設定すると SignedPutURL・Stat がそのエラーを返す（ストレージの障害の確認用）
	Err error
}

var _ Storage = (*Memory)(nil)

func NewMemory() *Memory {
	return &Memory{objects: map[string]ObjectInfo{}}
}

func (m *Memory) SignedPutURL(_ context.Context, bucket, object, contentType string, maxBytes int64, expiresAt time.Time) (SignedURL, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return SignedURL{}, m.Err
	}
	u := url.URL{Scheme: "https", Host: "storage.test", Path: "/" + bucket + "/" + object}
	return putURL(u.String(), putHeaders(contentType, maxBytes), expiresAt), nil
}

func (m *Memory) Stat(_ context.Context, bucket, object string) (ObjectInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return ObjectInfo{}, m.Err
	}
	info, ok := m.objects[bucket+"/"+object]
	if !ok {
		return ObjectInfo{}, ErrNotFound
	}
	return info, nil
}

// Put はブラウザがオブジェクトをアップロードしたことにする。
func (m *Memory) Put(bucket, object string, info ObjectInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objects[bucket+"/"+object] = info
}
