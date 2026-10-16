package service_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/auth"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/service"
)

const wrongPassword = "wrong password!!"

// failLogin は間違ったパスワードで n 回ログインし、すべて 401 になることを確かめる。
func failLogin(t *testing.T, f *loginFixture, addr string, n int) {
	t.Helper()
	for i := range n {
		_, err := f.login.Login(context.Background(), addr, wrongPassword, knownDevice)
		if apperr.CodeOf(err) != service.CodeInvalidCredentials {
			t.Fatalf("%d回目: %v", i+1, err)
		}
	}
}

// パスワードの入力ミスが10回続いたら15分間ログインを止める（AUTHN-6, ADR 0011）。
func TestLogin_入力ミスでのロック(t *testing.T) {
	ctx := context.Background()

	t.Run("10回続けて間違えたら正しいパスワードでも 429 になり、15分たてばログインできる", func(t *testing.T) {
		f := newLoginFixture(t)
		failLogin(t, f, email, auth.MaxPasswordFailures)
		_, err := f.login.Login(ctx, email, password, knownDevice)
		wantCode(t, err, apperr.CodeTooManyRequests)

		f.clock.Advance(auth.LoginLockDuration - time.Second)
		_, err = f.login.Login(ctx, email, password, knownDevice)
		wantCode(t, err, apperr.CodeTooManyRequests)

		f.clock.Advance(time.Second)
		res, err := f.login.Login(ctx, email, password, knownDevice)
		if err != nil || res.Session == nil {
			t.Fatalf("15分たってもログインできない: res=%+v err=%v", res, err)
		}
	})

	t.Run("ロックが解けたら回数は0から数える", func(t *testing.T) {
		f := newLoginFixture(t)
		failLogin(t, f, email, auth.MaxPasswordFailures)
		f.clock.Advance(auth.LoginLockDuration)
		failLogin(t, f, email, auth.MaxPasswordFailures-1)
		if _, err := f.login.Login(ctx, email, password, knownDevice); err != nil {
			t.Errorf("解除後の9回のミスで止まった: %v", err)
		}
	})

	t.Run("9回間違えても、一致したら回数を戻す", func(t *testing.T) {
		f := newLoginFixture(t)
		failLogin(t, f, email, auth.MaxPasswordFailures-1)
		if _, err := f.login.Login(ctx, email, password, knownDevice); err != nil {
			t.Fatal(err)
		}
		failLogin(t, f, email, auth.MaxPasswordFailures-1)
		if _, err := f.login.Login(ctx, email, password, knownDevice); err != nil {
			t.Errorf("一致した後の9回のミスで止まった: %v", err)
		}
	})

	t.Run("停止中でもパスワードが一致したら回数を戻す", func(t *testing.T) {
		f := newLoginFixture(t)
		f.setStatus(t, domain.UserStatusSuspended)
		failLogin(t, f, email, auth.MaxPasswordFailures-1)
		_, err := f.login.Login(ctx, email, password, knownDevice)
		wantCode(t, err, service.CodeAccountSuspended)
		failLogin(t, f, email, 1)
		_, err = f.login.Login(ctx, email, password, knownDevice)
		wantCode(t, err, service.CodeAccountSuspended)
	})

	t.Run("未登録・削除済みのアドレスも登録済みと同じように止める（AUTHN-2）", func(t *testing.T) {
		f := newLoginFixture(t)
		const deleted = "26238003@edu.cc.saga-u.ac.jp"
		f.signup(t, deleted, "")
		if err := f.gdb.Exec("UPDATE users SET status = ? WHERE email = ?", string(domain.UserStatusDeleted), deleted).Error; err != nil {
			t.Fatal(err)
		}
		var errs []error
		for _, addr := range []string{email, "26238002@edu.cc.saga-u.ac.jp", deleted} {
			failLogin(t, f, addr, auth.MaxPasswordFailures)
			_, err := f.login.Login(ctx, addr, password, knownDevice)
			errs = append(errs, err)
		}
		for _, err := range errs {
			wantCode(t, err, apperr.CodeTooManyRequests)
			if err.Error() != errs[0].Error() {
				t.Errorf("応答が違う: %v / %v", err, errs[0])
			}
		}
	})

	t.Run("アドレスの大文字・空白の違いは同じアドレスとして数え、別のアドレスは止めない", func(t *testing.T) {
		f := newLoginFixture(t)
		failLogin(t, f, " 26238001@EDU.CC.SAGA-U.AC.JP", auth.MaxPasswordFailures)
		_, err := f.login.Login(ctx, email, password, knownDevice)
		wantCode(t, err, apperr.CodeTooManyRequests)
		_, err = f.login.Login(ctx, "26238002@edu.cc.saga-u.ac.jp", wrongPassword, "")
		wantCode(t, err, service.CodeInvalidCredentials)
	})

	t.Run("同時に入力しても照合するのは10回まで", func(t *testing.T) {
		f := newLoginFixture(t)
		var (
			mu     sync.Mutex
			counts = map[apperr.Code]int{}
			wg     sync.WaitGroup
		)
		for range 15 {
			wg.Go(func() {
				_, err := f.login.Login(ctx, email, wrongPassword, knownDevice)
				mu.Lock()
				counts[apperr.CodeOf(err)]++
				mu.Unlock()
			})
		}
		wg.Wait()
		if counts[service.CodeInvalidCredentials] != auth.MaxPasswordFailures || counts[apperr.CodeTooManyRequests] != 5 {
			t.Errorf("counts=%v", counts)
		}
	})
}

// パスワード再設定に成功したら入力ミスの回数を戻すが、ロックは期限まで続く（AUTHN-4, AUTHN-6, ADR 0011）。
func TestPasswordReset_入力ミスの回数(t *testing.T) {
	ctx := context.Background()

	t.Run("回数を戻す", func(t *testing.T) {
		f := newResetFixture(t)
		failLogin(t, f.loginFixture, email, auth.MaxPasswordFailures-1)
		if _, err := f.reset.Complete(ctx, f.verified(t), newPassword, knownDevice); err != nil {
			t.Fatal(err)
		}
		failLogin(t, f.loginFixture, email, auth.MaxPasswordFailures-1)
		if _, err := f.login.Login(ctx, email, newPassword, knownDevice); err != nil {
			t.Errorf("再設定の後の9回のミスで止まった: %v", err)
		}
	})

	t.Run("ロック中ならロックは期限まで続く", func(t *testing.T) {
		f := newResetFixture(t)
		failLogin(t, f.loginFixture, email, auth.MaxPasswordFailures)
		if _, err := f.reset.Complete(ctx, f.verified(t), newPassword, knownDevice); err != nil {
			t.Fatal(err)
		}
		_, err := f.login.Login(ctx, email, newPassword, knownDevice)
		wantCode(t, err, apperr.CodeTooManyRequests)
		f.clock.Advance(auth.LoginLockDuration)
		if _, err := f.login.Login(ctx, email, newPassword, knownDevice); err != nil {
			t.Errorf("期限が過ぎてもログインできない: %v", err)
		}
	})
}
