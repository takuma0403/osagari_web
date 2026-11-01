package mail

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"mime"
	"net"
	netmail "net/mail"
	"net/smtp"
	"strings"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/clock"
)

// SMTPConfig は SMTP サーバーへの接続設定。値の一覧は .env.example。
type SMTPConfig struct {
	Host string
	Port string
	// Username が空なら認証しない（ローカルの Mailpit）
	Username string
	Password string
	// From は差出人。"addr@example.com" または "名前 <addr@example.com>"
	From string
	// StartTLS が true なら STARTTLS を必須にする。Gmail（587）は true、Mailpit は false（AUTHN-7）
	StartTLS bool
	// Timeout は接続から送信完了までの上限。0 なら 30 秒
	Timeout time.Duration
}

// SMTP は SMTPConfig のサーバーへ送る Mailer。
type SMTP struct {
	cfg   SMTPConfig
	from  *netmail.Address
	clock clock.Clock
}

var _ Mailer = (*SMTP)(nil)

// NewSMTP は SMTP の Mailer を返す。Date ヘッダーの時刻は clk から取る。
func NewSMTP(cfg SMTPConfig, clk clock.Clock) (*SMTP, error) {
	if cfg.Host == "" || cfg.Port == "" {
		return nil, fmt.Errorf("mail: SMTP のホストとポートが必要です")
	}
	from, err := netmail.ParseAddress(cfg.From)
	if err != nil {
		return nil, fmt.Errorf("mail: 差出人のアドレスが不正です: %w", err)
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	return &SMTP{cfg: cfg, from: from, clock: clk}, nil
}

func (s *SMTP) Send(ctx context.Context, msg Message) error {
	if err := msg.validate(); err != nil {
		return err
	}
	data, err := s.build(msg)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, s.cfg.Timeout)
	defer cancel()
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(s.cfg.Host, s.cfg.Port))
	if err != nil {
		return fmt.Errorf("mail: SMTP に接続できません: %w", err)
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return err
		}
	}

	c, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		return fmt.Errorf("mail: SMTP の応答が不正です: %w", err)
	}
	defer c.Close()

	if s.cfg.StartTLS {
		if ok, _ := c.Extension("STARTTLS"); !ok {
			return fmt.Errorf("mail: サーバーが STARTTLS に対応していません")
		}
		if err := c.StartTLS(&tls.Config{ServerName: s.cfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("mail: STARTTLS に失敗しました: %w", err)
		}
	}
	if s.cfg.Username != "" {
		if err := c.Auth(smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)); err != nil {
			return fmt.Errorf("mail: SMTP の認証に失敗しました: %w", err)
		}
	}
	if err := c.Mail(s.from.Address); err != nil {
		return fmt.Errorf("mail: 差出人を受け付けられませんでした: %w", err)
	}
	if err := c.Rcpt(msg.To); err != nil {
		return fmt.Errorf("mail: 宛先を受け付けられませんでした: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("mail: 本文を送れません: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("mail: 本文を送れません: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("mail: 本文を送れません: %w", err)
	}
	return c.Quit()
}

// build はヘッダーと本文を組み立てる。件名は日本語を含むので MIME エンコードし、本文は base64 で送る。
func (s *SMTP) build(msg Message) ([]byte, error) {
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		return nil, err
	}
	domain := s.from.Address[strings.LastIndex(s.from.Address, "@")+1:]

	var b bytes.Buffer
	header := func(k, v string) { fmt.Fprintf(&b, "%s: %s\r\n", k, v) }
	header("From", s.from.String())
	header("To", msg.To)
	header("Subject", mime.BEncoding.Encode("UTF-8", msg.Subject))
	header("Date", s.clock.Now().Format(time.RFC1123Z))
	header("Message-ID", fmt.Sprintf("<%s@%s>", hex.EncodeToString(id), domain))
	header("MIME-Version", "1.0")
	header("Content-Type", "text/plain; charset=UTF-8")
	header("Content-Transfer-Encoding", "base64")
	b.WriteString("\r\n")

	// base64 は1行76文字以内で折り返す（RFC 2045）
	enc := base64.StdEncoding.EncodeToString([]byte(msg.Body))
	for len(enc) > 76 {
		b.WriteString(enc[:76] + "\r\n")
		enc = enc[76:]
	}
	b.WriteString(enc + "\r\n")
	return b.Bytes(), nil
}
