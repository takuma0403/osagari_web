// Mailpit に実際に送って届くかの確認（AUTHN-7）。
// TEST_SMTP_HOST を設定したときだけ、compose の mail に対して実行する（未設定ならスキップ）：
//
//	docker compose run --rm -e TEST_SMTP_HOST=mail -e TEST_MAILPIT_URL=http://mail:8025 api go test -v ./internal/mail/...
//
// ビルドタグで分けると gopls がこのファイルを解析しないため、環境変数で切り替える。
package mail_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/clock"
	"github.com/takuma0403/osagari_web/server/internal/mail"
)

func TestMailpitに届く(t *testing.T) {
	host := os.Getenv("TEST_SMTP_HOST")
	if host == "" {
		t.Skip("TEST_SMTP_HOST が未設定")
	}
	port := os.Getenv("TEST_SMTP_PORT")
	if port == "" {
		port = "1025"
	}
	m, err := mail.NewSMTP(mail.SMTPConfig{Host: host, Port: port, From: "お下がりフリマ <noreply@example.com>"}, clock.System())
	if err != nil {
		t.Fatal(err)
	}

	// 他のメールと見分けるため、件名に乱数を入れる
	b := make([]byte, 8)
	rand.Read(b)
	marker := hex.EncodeToString(b)
	subject := "【お下がりフリマ】テスト " + marker
	if err := m.Send(context.Background(), mail.Message{To: "26238001@edu.cc.saga-u.ac.jp", Subject: subject, Body: "テストメールです。"}); err != nil {
		t.Fatalf("送信できない: %v", err)
	}

	api := os.Getenv("TEST_MAILPIT_URL")
	if api == "" {
		return
	}
	// Mailpit の検索 API で、届いたメールの件名を確かめる
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		got, err := searchSubjects(api, marker)
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range got {
			if s == subject {
				return
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("Mailpit に件名 %q のメールが届いていない", subject)
}

func searchSubjects(api, query string) ([]string, error) {
	resp, err := http.Get(api + "/api/v1/search?query=" + url.QueryEscape(query))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var body struct {
		Messages []struct {
			Subject string `json:"Subject"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	subjects := make([]string, 0, len(body.Messages))
	for _, m := range body.Messages {
		subjects = append(subjects, m.Subject)
	}
	return subjects, nil
}
