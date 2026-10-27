package storage

import (
	"context"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"
)

// Memory はテスト用の Storage。オブジェクトをメモリに持ち、Put でアップロードしたことにする。並行に使ってよい。
type Memory struct {
	mu      sync.Mutex
	objects map[string]memoryObject
	// Err を設定するとすべての操作がそのエラーを返す（ストレージの障害の確認用）
	Err error
}

type memoryObject struct {
	info    ObjectInfo
	created time.Time
}

var _ Storage = (*Memory)(nil)

func NewMemory() *Memory {
	return &Memory{objects: map[string]memoryObject{}}
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

func (m *Memory) SignedGetURL(_ context.Context, bucket, object string, expiresAt time.Time) (SignedURL, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return SignedURL{}, m.Err
	}
	u := url.URL{Scheme: "https", Host: "storage.test", Path: "/" + bucket + "/" + object}
	return getURL(u.String(), expiresAt), nil
}

func (m *Memory) Stat(_ context.Context, bucket, object string) (ObjectInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return ObjectInfo{}, m.Err
	}
	o, ok := m.objects[bucket+"/"+object]
	if !ok {
		return ObjectInfo{}, ErrNotFound
	}
	return o.info, nil
}

func (m *Memory) Delete(_ context.Context, bucket, object string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return m.Err
	}
	key := bucket + "/" + object
	if _, ok := m.objects[key]; !ok {
		return ErrNotFound
	}
	delete(m.objects, key)
	return nil
}

// List は名前順に返す。
func (m *Memory) List(_ context.Context, bucket string) ([]ObjectEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return nil, m.Err
	}
	var out []ObjectEntry
	for key, o := range m.objects {
		if name, ok := strings.CutPrefix(key, bucket+"/"); ok {
			out = append(out, ObjectEntry{Name: name, Created: o.created})
		}
	}
	slices.SortFunc(out, func(a, b ObjectEntry) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

// Put はブラウザがオブジェクトをアップロードしたことにする。作成日時はゼロ値（一覧では十分に古いものとして扱われる）。
func (m *Memory) Put(bucket, object string, info ObjectInfo) {
	m.PutAt(bucket, object, info, time.Time{})
}

// PutAt は created にアップロードしたことにする。
func (m *Memory) PutAt(bucket, object string, info ObjectInfo, created time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objects[bucket+"/"+object] = memoryObject{info: info, created: created}
}

// Has はオブジェクトがあるかを返す。
func (m *Memory) Has(bucket, object string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.objects[bucket+"/"+object]
	return ok
}
