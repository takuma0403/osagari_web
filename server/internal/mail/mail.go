// Package mail はメールの送信を抽象化する（AUTHN-7）。
// 本番は Gmail SMTP（587 / STARTTLS）、ローカルは Mailpit、テストは Recorder を差し込む。
package mail

import (
	"context"
	"errors"
	"strings"
)

// Message は送信するメール1通。本文はプレーンテキスト（UTF-8）。
type Message struct {
	To      string
	Subject string
	Body    string
}

// Mailer はメールを送る。service はこのインターフェース越しに使う。
type Mailer interface {
	Send(ctx context.Context, msg Message) error
}

// ErrInvalidMessage は宛先・件名が空、または改行を含む（ヘッダーの差し込みになる）ときに返す。
var ErrInvalidMessage = errors.New("mail: 宛先または件名が不正です")

func (m Message) validate() error {
	if m.To == "" || m.Subject == "" {
		return ErrInvalidMessage
	}
	if strings.ContainsAny(m.To, "\r\n") || strings.ContainsAny(m.Subject, "\r\n") {
		return ErrInvalidMessage
	}
	return nil
}
