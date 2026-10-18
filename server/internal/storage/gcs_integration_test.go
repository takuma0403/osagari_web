// fake-gcs-server に対して「署名付きURLの発行 → PUT → Stat」ができるかの検証（STORAGE）。
// TEST_STORAGE_EMULATOR_HOST を設定したときだけ、compose の storage に対して実行する（未設定ならスキップ）：
//
//	docker compose run --rm -e TEST_STORAGE_EMULATOR_HOST=storage:4443 api go test -v ./internal/storage/...
//
// ビルドタグで分けると gopls がこのファイルを解析しないため、環境変数で切り替える。
// fake-gcs-server は -public-host（compose では localhost:4443）と一致する Host のときだけパス形式の PUT を受け付けるため、
// URL にはブラウザ向けのホストを入れ、接続先だけエミュレータに向ける。
package storage_test

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/takuma0403/osagari_web/server/internal/storage"
)

// bucket は compose の storage-init が作るバケット。
const bucket = "student-cards"

// publicHost は compose の fake-gcs-server の -public-host（ブラウザから届くホスト）。
const publicHost = "localhost:4443"

func TestGCSEmulator(t *testing.T) {
	host := os.Getenv("TEST_STORAGE_EMULATOR_HOST")
	if host == "" {
		t.Skip("TEST_STORAGE_EMULATOR_HOST が未設定")
	}
	t.Setenv("STORAGE_EMULATOR_HOST", host)
	ctx := context.Background()
	g, err := storage.NewGCS(ctx, storage.GCSConfig{EmulatorPublicHost: publicHost})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { g.Close() })

	object := "integration-test/" + uuid.NewString()
	if _, err := g.Stat(ctx, bucket, object); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("アップロード前の Stat: err = %v, want ErrNotFound", err)
	}

	u, err := g.SignedPutURL(ctx, bucket, object, "image/jpeg", time.Now().Add(15*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if u.Method != http.MethodPut || u.Headers["Content-Type"] != "image/jpeg" {
		t.Fatalf("SignedURL = %+v", u)
	}
	if !strings.HasPrefix(u.URL, "http://"+publicHost+"/"+bucket+"/") || !strings.Contains(u.URL, "X-Goog-Signature=") {
		t.Fatalf("URL = %s", u.URL)
	}

	body := "jpeg-bytes"
	req, err := http.NewRequestWithContext(ctx, u.Method, u.URL, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.URL.Host = host
	for k, v := range u.Headers {
		req.Header.Set(k, v)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PUT status = %d", res.StatusCode)
	}

	info, err := g.Stat(ctx, bucket, object)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size != int64(len(body)) || info.ContentType != "image/jpeg" {
		t.Fatalf("Stat = %+v", info)
	}
}
