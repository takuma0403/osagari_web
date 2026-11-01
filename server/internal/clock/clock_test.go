package clock_test

import (
	"sync"
	"testing"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/clock"
)

func TestSystemはUTCを返す(t *testing.T) {
	if loc := clock.System().Now().Location(); loc != time.UTC {
		t.Fatalf("location=%v want=UTC", loc)
	}
}

func TestFake(t *testing.T) {
	base := time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)
	f := clock.NewFake(base)

	if got := f.Now(); !got.Equal(base) {
		t.Fatalf("Now=%v want=%v", got, base)
	}
	f.Advance(48 * time.Hour)
	if want := base.Add(48 * time.Hour); !f.Now().Equal(want) {
		t.Fatalf("Advance 後 Now=%v want=%v", f.Now(), want)
	}
	other := time.Date(2027, 3, 31, 0, 0, 0, 0, time.UTC)
	f.Set(other)
	if !f.Now().Equal(other) {
		t.Fatalf("Set 後 Now=%v want=%v", f.Now(), other)
	}
}

func TestFakeは並行に進めてもずれない(t *testing.T) {
	base := time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)
	f := clock.NewFake(base)
	const n = 100
	var wg sync.WaitGroup
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f.Advance(time.Second)
			_ = f.Now()
		}()
	}
	wg.Wait()
	if want := base.Add(n * time.Second); !f.Now().Equal(want) {
		t.Fatalf("Now=%v want=%v", f.Now(), want)
	}
}
