package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/auth"
	"github.com/takuma0403/osagari_web/server/internal/repository"
	"github.com/takuma0403/osagari_web/server/internal/service"
)

// 古い入力ミスの記録を消す（AUTHN-6, ADR 0011）。止めている間の記録は消さない。
func TestDeleteStaleLoginFailures(t *testing.T) {
	ctx := context.Background()
	f := newLoginFixture(t)
	cleanup := service.NewAuthCleanupService(service.AuthCleanupDeps{
		Failures: repository.NewLoginFailureRepository(f.gdb),
		Clock:    f.clock,
	})
	const (
		few    = "26238002@edu.cc.saga-u.ac.jp"
		locked = "26238003@edu.cc.saga-u.ac.jp"
	)
	failLogin(t, f, few, 3)
	failLogin(t, f, locked, auth.MaxPasswordFailures)

	f.clock.Advance(auth.LoginFailureRetention - time.Second)
	failLogin(t, f, "26238004@edu.cc.saga-u.ac.jp", 1) // 新しい記録
	f.clock.Advance(time.Second)
	if n, err := cleanup.DeleteStaleLoginFailures(ctx); err != nil || n != 2 {
		t.Fatalf("消した件数: n=%d err=%v（24時間更新のない2件を消す）", n, err)
	}
	if got := f.count(t, "login_failures"); got != 1 {
		t.Errorf("残った記録=%d want=1", got)
	}
	// 消した記録の回数は0から数え直す
	failLogin(t, f, few, auth.MaxPasswordFailures-1)
	_, err := f.login.Login(ctx, few, password, "")
	wantCode(t, err, service.CodeInvalidCredentials) // 3回分が残っていれば、ここで止まって 429 になる
}

func TestDeleteStaleLoginFailures_止めている間は消さない(t *testing.T) {
	ctx := context.Background()
	f := newLoginFixture(t)
	failures := repository.NewLoginFailureRepository(f.gdb)
	failLogin(t, f, email, auth.MaxPasswordFailures)
	// 止めている間に、更新が24時間より前の記録として消しにかかる
	if n, err := failures.DeleteStale(ctx, f.clock.Now().Add(time.Second), f.clock.Now()); err != nil || n != 0 {
		t.Fatalf("止めている間の記録を消した: n=%d err=%v", n, err)
	}
	_, err := f.login.Login(ctx, email, password, knownDevice)
	wantCode(t, err, apperr.CodeTooManyRequests)
}
