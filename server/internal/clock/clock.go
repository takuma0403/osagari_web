// Package clock は現在時刻の取得を抽象化する。期限（48時間・10分など）の判定をテストで固定できるように、
// アプリのコードは time.Now() を直接呼ばず Clock を使う。
package clock

import (
	"sync"
	"time"
)

// Clock は現在時刻を返す。
type Clock interface {
	Now() time.Time
}

// System は実時計を返す。時刻は UTC で返す。
func System() Clock {
	return systemClock{}
}

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now().UTC()
}

// Fake はテスト用の固定時計。Set・Advance でしか進まない。並行に使ってよい。
type Fake struct {
	mu  sync.Mutex
	now time.Time
}

var _ Clock = (*Fake)(nil)

// NewFake は t を現在時刻とする固定時計を返す。
func NewFake(t time.Time) *Fake {
	return &Fake{now: t}
}

func (f *Fake) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

// Set は現在時刻を t にする。
func (f *Fake) Set(t time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = t
}

// Advance は現在時刻を d だけ進める。
func (f *Fake) Advance(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = f.now.Add(d)
}
