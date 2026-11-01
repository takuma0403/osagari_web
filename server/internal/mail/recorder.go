package mail

import (
	"context"
	"sync"
)

// Recorder はテスト用の Mailer。送らずに記録する。並行に使ってよい。
type Recorder struct {
	mu   sync.Mutex
	sent []Message
	// Err を設定すると Send がそのエラーを返す（送信失敗の確認用）。記録はしない
	Err error
}

var _ Mailer = (*Recorder)(nil)

func (r *Recorder) Send(_ context.Context, msg Message) error {
	if err := msg.validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return r.Err
	}
	r.sent = append(r.sent, msg)
	return nil
}

// Messages は記録したメールを送った順に返す。
func (r *Recorder) Messages() []Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Message(nil), r.sent...)
}

// Last は最後に記録したメールを返す。1通もなければ false。
func (r *Recorder) Last() (Message, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.sent) == 0 {
		return Message{}, false
	}
	return r.sent[len(r.sent)-1], true
}
