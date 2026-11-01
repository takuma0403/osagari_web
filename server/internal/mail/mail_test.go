package mail

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"mime"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/clock"
)

func TestMessageの検証(t *testing.T) {
	cases := []struct {
		name string
		msg  Message
		ok   bool
	}{
		{"正常", Message{To: "a@example.com", Subject: "件名", Body: "本文"}, true},
		{"宛先が空", Message{Subject: "件名"}, false},
		{"件名が空", Message{To: "a@example.com"}, false},
		{"宛先に改行", Message{To: "a@example.com\r\nBcc: b@example.com", Subject: "件名"}, false},
		{"件名に改行", Message{To: "a@example.com", Subject: "件名\nBcc: b@example.com"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.validate()
			if tc.ok != (err == nil) {
				t.Fatalf("err=%v", err)
			}
			if err != nil && !errors.Is(err, ErrInvalidMessage) {
				t.Fatalf("ErrInvalidMessage ではない: %v", err)
			}
		})
	}
}

func TestNewSMTPの設定チェック(t *testing.T) {
	clk := clock.NewFake(time.Now())
	if _, err := NewSMTP(SMTPConfig{Port: "1025", From: "a@example.com"}, clk); err == nil {
		t.Error("ホストが空でもエラーにならない")
	}
	if _, err := NewSMTP(SMTPConfig{Host: "mail", Port: "1025", From: "not-an-address"}, clk); err == nil {
		t.Error("差出人が不正でもエラーにならない")
	}
	if _, err := NewSMTP(SMTPConfig{Host: "mail", Port: "1025", From: "お下がりフリマ <noreply@example.com>"}, clk); err != nil {
		t.Errorf("表示名つきの差出人を受け付けない: %v", err)
	}
}

func TestSMTPのメール組み立て(t *testing.T) {
	now := time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)
	s, err := NewSMTP(SMTPConfig{Host: "mail", Port: "1025", From: "お下がりフリマ <noreply@example.com>"}, clock.NewFake(now))
	if err != nil {
		t.Fatal(err)
	}
	body := strings.Repeat("認証コードは 123456 です。", 10)
	data, err := s.build(Message{To: "26238001@edu.cc.saga-u.ac.jp", Subject: "【お下がりフリマ】認証コード", Body: body})
	if err != nil {
		t.Fatal(err)
	}

	r := textproto.NewReader(bufio.NewReader(bytes.NewReader(data)))
	h, err := r.ReadMIMEHeader()
	if err != nil {
		t.Fatal(err)
	}
	subject, err := new(mime.WordDecoder).DecodeHeader(h.Get("Subject"))
	if err != nil || subject != "【お下がりフリマ】認証コード" {
		t.Errorf("Subject=%q err=%v", subject, err)
	}
	if got := h.Get("To"); got != "26238001@edu.cc.saga-u.ac.jp" {
		t.Errorf("To=%q", got)
	}
	if got := h.Get("Date"); got != now.Format(time.RFC1123Z) {
		t.Errorf("Date=%q（時刻は clock から取る）", got)
	}
	if got := h.Get("Message-Id"); !strings.HasSuffix(got, "@example.com>") {
		t.Errorf("Message-ID=%q", got)
	}

	var enc strings.Builder
	for {
		line, err := r.ReadLine()
		if err != nil {
			break
		}
		if len(line) > 76 {
			t.Errorf("本文の行が76文字を超える: %d", len(line))
		}
		enc.WriteString(line)
	}
	decoded, err := base64.StdEncoding.DecodeString(enc.String())
	if err != nil || string(decoded) != body {
		t.Errorf("本文が復元できない: err=%v", err)
	}
}

func TestRecorder(t *testing.T) {
	ctx := context.Background()
	var r Recorder
	if _, ok := r.Last(); ok {
		t.Fatal("空なのに Last が返る")
	}
	for _, subject := range []string{"1通目", "2通目"} {
		if err := r.Send(ctx, Message{To: "a@example.com", Subject: subject}); err != nil {
			t.Fatal(err)
		}
	}
	if got := r.Messages(); len(got) != 2 || got[0].Subject != "1通目" {
		t.Errorf("Messages=%v", got)
	}
	if last, _ := r.Last(); last.Subject != "2通目" {
		t.Errorf("Last=%v", last)
	}

	r.Err = errors.New("送信失敗")
	if err := r.Send(ctx, Message{To: "a@example.com", Subject: "3通目"}); err == nil {
		t.Error("Err を設定しても失敗しない")
	}
	if len(r.Messages()) != 2 {
		t.Error("失敗した送信が記録されている")
	}
}
