// S3 互換ストレージ（ローカルは RustFS。本番は Cloudflare R2）に対して、署名付きURLの条件が守られるかの検証（STORAGE, ADR 0051・0052）。
// TEST_S3_ENDPOINT を設定したときだけ、compose の s3 に対して実行する（未設定ならスキップ）：
//
//	docker compose run --rm -e TEST_S3_ENDPOINT=http://s3:9000 api go test -v ./internal/storage/...
//
// RustFS は R2 と同じく署名を検証するので、署名に含めた Content-Type・Content-Length・If-None-Match と違う PUT は拒まれる。
package storage_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/takuma0403/osagari_web/server/internal/storage"
)

// s3Bucket は compose の s3-init（CI では dev/s3-init.sh）が作るバケット。
const s3Bucket = "item-images"

func TestS3(t *testing.T) {
	endpoint := os.Getenv("TEST_S3_ENDPOINT")
	if endpoint == "" {
		t.Skip("TEST_S3_ENDPOINT が未設定")
	}
	ctx := context.Background()
	// アクセスキーは compose の RustFS のローカル専用の値
	s, err := storage.NewS3(storage.S3Config{
		Endpoint:        endpoint,
		Region:          "us-east-1",
		AccessKeyID:     "local-access-key",
		SecretAccessKey: "local-secret-key",
	})
	if err != nil {
		t.Fatal(err)
	}

	body := "jpeg-bytes"
	size := int64(len(body))
	opts := storage.PutOptions{ContentType: "image/jpeg", MaxBytes: 5 << 20, Size: size}
	expires := time.Now().Add(15 * time.Minute)

	// put は署名付きURLに body を PUT し、ステータスを返す。headers はそのまま付け、edit で書き換える
	put := func(t *testing.T, u storage.SignedURL, body string, edit func(h http.Header)) int {
		t.Helper()
		req, err := http.NewRequestWithContext(ctx, u.Method, u.URL, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range u.Headers {
			req.Header.Set(k, v)
		}
		if edit != nil {
			edit(req.Header)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		return res.StatusCode
	}

	t.Run("大きさが範囲外なら発行しない", func(t *testing.T) {
		for _, size := range []int64{0, 5<<20 + 1} {
			o := opts
			o.Size = size
			if _, err := s.SignedPutURL(ctx, s3Bucket, "integration-test/"+uuid.NewString(), o, expires); err == nil {
				t.Errorf("size=%d: エラーにならない", size)
			}
		}
	})

	t.Run("署名と違う PUT は拒む", func(t *testing.T) {
		cases := []struct {
			name string
			body string
			edit func(h http.Header)
		}{
			{"形式が違う", body, func(h http.Header) { h.Set("Content-Type", "image/webp") }},
			{"大きさが違う", body + "x", nil},
			{"If-None-Match がない", body, func(h http.Header) { h.Del("If-None-Match") }},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				object := "integration-test/" + uuid.NewString()
				u, err := s.SignedPutURL(ctx, s3Bucket, object, opts, expires)
				if err != nil {
					t.Fatal(err)
				}
				if got := put(t, u, c.body, c.edit); got != http.StatusForbidden {
					t.Fatalf("PUT status = %d, want 403", got)
				}
				if _, err := s.Stat(ctx, s3Bucket, object); !errors.Is(err, storage.ErrNotFound) {
					t.Fatalf("拒んだ後の Stat: err = %v, want ErrNotFound", err)
				}
			})
		}
	})

	object := "integration-test/" + uuid.NewString()
	if _, err := s.Stat(ctx, s3Bucket, object); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("アップロード前の Stat: err = %v, want ErrNotFound", err)
	}
	u, err := s.SignedPutURL(ctx, s3Bucket, object, opts, expires)
	if err != nil {
		t.Fatal(err)
	}
	if u.Method != http.MethodPut || len(u.Headers) != 2 || u.Headers["Content-Type"] != "image/jpeg" || u.Headers["If-None-Match"] != "*" {
		t.Fatalf("SignedURL = %+v", u)
	}
	if !strings.HasPrefix(u.URL, endpoint+"/"+s3Bucket+"/") || !strings.Contains(u.URL, "X-Amz-Signature=") ||
		!strings.Contains(u.URL, "content-length%3Bcontent-type") || !strings.Contains(u.URL, "if-none-match") {
		t.Fatalf("URL = %s", u.URL)
	}
	if got := put(t, u, body, nil); got != http.StatusOK {
		t.Fatalf("PUT status = %d", got)
	}
	// 登録した後に同じURLで画像を差し替えさせない
	if got := put(t, u, body, nil); got != http.StatusPreconditionFailed {
		t.Fatalf("2回目の PUT status = %d, want 412", got)
	}

	info, err := s.Stat(ctx, s3Bucket, object)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size != size || info.ContentType != "image/jpeg" {
		t.Fatalf("Stat = %+v", info)
	}

	gu, err := s.SignedGetURL(ctx, s3Bucket, object, expires)
	if err != nil {
		t.Fatal(err)
	}
	if gu.Method != http.MethodGet || len(gu.Headers) != 0 || !strings.HasPrefix(gu.URL, endpoint+"/"+s3Bucket+"/") {
		t.Fatalf("SignedGetURL = %+v", gu)
	}
	gres, err := http.Get(gu.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer gres.Body.Close()
	got, err := io.ReadAll(gres.Body)
	if err != nil {
		t.Fatal(err)
	}
	if gres.StatusCode != http.StatusOK || string(got) != body {
		t.Fatalf("GET status = %d body = %q", gres.StatusCode, got)
	}

	list, err := s.List(ctx, s3Bucket)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range list {
		if e.Name == object {
			found = true
			if e.Created.IsZero() {
				t.Errorf("作成日時が空: %+v", e)
			}
		}
	}
	if !found {
		t.Fatalf("一覧に %s がない", object)
	}
	if err := s.Delete(ctx, s3Bucket, object); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Stat(ctx, s3Bucket, object); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("削除後の Stat: err = %v, want ErrNotFound", err)
	}
	if err := s.Delete(ctx, s3Bucket, object); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("2回目の Delete: err = %v, want ErrNotFound", err)
	}
}
