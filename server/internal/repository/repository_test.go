package repository_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/repository"
	"github.com/takuma0403/osagari_web/server/internal/testutil/testdb"
)

var t0 = time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)

const email = "26238001@edu.cc.saga-u.ac.jp"

func newUser(id, email, number string) *domain.User {
	dept := "238"
	return &domain.User{
		ID: id, Email: email, StudentNumber: number, AdmissionYear: 2026,
		DepartmentCode: "238", Department: &dept,
		ExpiresOn: time.Date(2030, 3, 31, 0, 0, 0, 0, time.UTC),
		Status:    domain.UserStatusActive, SellerStatus: domain.SellerStatusNone,
		PasswordHash: "hash", CodeVerifiedAt: t0, CreatedAt: t0, UpdatedAt: t0,
	}
}

func TestUserRepository(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	users := repository.NewUserRepository(gdb)

	u := newUser("", email, "26238001")
	if err := users.Create(ctx, u); err != nil {
		t.Fatal(err)
	}
	if u.ID == "" {
		t.Fatal("ID が採番されていない")
	}

	got, err := users.FindByID(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Email != email || *got.Department != "238" || got.ExpiresOn.Format("2006-01-02") != "2030-03-31" ||
		!got.CodeVerifiedAt.Equal(t0) || got.Status != domain.UserStatusActive {
		t.Errorf("got=%+v", got)
	}

	if ok, err := users.ExistsByEmail(ctx, email); err != nil || !ok {
		t.Errorf("登録済みのアドレス: ok=%v err=%v", ok, err)
	}
	if ok, err := users.ExistsByEmail(ctx, "26238002@edu.cc.saga-u.ac.jp"); err != nil || ok {
		t.Errorf("未登録のアドレス: ok=%v err=%v", ok, err)
	}
	if err := users.Create(ctx, newUser("", email, "26238009")); !errors.Is(err, repository.ErrDuplicate) {
		t.Errorf("同じアドレスで ErrDuplicate にならない: %v", err)
	}
	if _, err := users.FindByID(ctx, "no-such-user"); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("ErrNotFound にならない: %v", err)
	}

	if got, err := users.FindByEmail(ctx, email); err != nil || got.ID != u.ID {
		t.Errorf("FindByEmail: got=%+v err=%v", got, err)
	}
	if _, err := users.FindByEmail(ctx, "26238002@edu.cc.saga-u.ac.jp"); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("FindByEmail が ErrNotFound にならない: %v", err)
	}

	later := t0.Add(31 * 24 * time.Hour)
	if err := users.UpdateCodeVerifiedAt(ctx, u.ID, later); err != nil {
		t.Fatal(err)
	}
	if got, _ := users.FindByID(ctx, u.ID); !got.CodeVerifiedAt.Equal(later) || !got.UpdatedAt.Equal(later) {
		t.Errorf("UpdateCodeVerifiedAt: code_verified_at=%s updated_at=%s", got.CodeVerifiedAt, got.UpdatedAt)
	}
	if err := users.UpdateCodeVerifiedAt(ctx, "no-such-user", later); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("UpdateCodeVerifiedAt が ErrNotFound にならない: %v", err)
	}

	reset := later.Add(time.Hour)
	if err := users.UpdatePassword(ctx, u.ID, "new-hash", reset); err != nil {
		t.Fatal(err)
	}
	if got, _ := users.FindByID(ctx, u.ID); got.PasswordHash != "new-hash" || !got.CodeVerifiedAt.Equal(reset) || !got.UpdatedAt.Equal(reset) {
		t.Errorf("UpdatePassword: got=%+v", got)
	}
	if err := users.UpdatePassword(ctx, "no-such-user", "new-hash", reset); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("UpdatePassword が ErrNotFound にならない: %v", err)
	}
}

func newCode(createdAt time.Time) *domain.VerificationCode {
	return &domain.VerificationCode{
		Email: email, Purpose: domain.VerificationPurposeSignup, CodeHash: "hash",
		ExpiresAt: createdAt.Add(10 * time.Minute), CreatedAt: createdAt, UpdatedAt: createdAt,
	}
}

func TestVerificationCodeRepository(t *testing.T) {
	ctx := context.Background()
	const maxAttempts = 5

	t.Run("いちばん新しい使えるコードを返し、再送で古いコードを無効にする", func(t *testing.T) {
		codes := repository.NewVerificationCodeRepository(testdb.New(t))
		old := newCode(t0)
		if err := codes.Create(ctx, old); err != nil {
			t.Fatal(err)
		}
		if err := codes.InvalidateActive(ctx, email, domain.VerificationPurposeSignup, t0.Add(time.Minute)); err != nil {
			t.Fatal(err)
		}
		latest := newCode(t0.Add(time.Minute))
		if err := codes.Create(ctx, latest); err != nil {
			t.Fatal(err)
		}
		got, err := codes.FindLatestActive(ctx, email, domain.VerificationPurposeSignup, maxAttempts, t0.Add(2*time.Minute))
		if err != nil || got.ID != latest.ID {
			t.Fatalf("got=%+v err=%v", got, err)
		}
		if err := codes.Consume(ctx, old.ID, t0.Add(2*time.Minute)); !errors.Is(err, repository.ErrConflict) {
			t.Errorf("無効にしたコードを使えてしまう: %v", err)
		}
		// 用途が違えば見つからない
		if _, err := codes.FindLatestActive(ctx, email, domain.VerificationPurposePasswordReset, maxAttempts, t0); !errors.Is(err, repository.ErrNotFound) {
			t.Errorf("別の用途のコードが見つかる: %v", err)
		}
	})

	t.Run("期限切れのコードは見つからず、使えない", func(t *testing.T) {
		codes := repository.NewVerificationCodeRepository(testdb.New(t))
		c := newCode(t0)
		if err := codes.Create(ctx, c); err != nil {
			t.Fatal(err)
		}
		expired := t0.Add(10 * time.Minute)
		if _, err := codes.FindLatestActive(ctx, email, domain.VerificationPurposeSignup, maxAttempts, expired); !errors.Is(err, repository.ErrNotFound) {
			t.Errorf("期限切れが見つかる: %v", err)
		}
		if _, err := codes.ReserveAttempt(ctx, c.ID, maxAttempts, expired); !errors.Is(err, repository.ErrConflict) {
			t.Errorf("期限切れで回数を数えられる: %v", err)
		}
		if err := codes.Consume(ctx, c.ID, expired); !errors.Is(err, repository.ErrConflict) {
			t.Errorf("期限切れを使える: %v", err)
		}
	})

	t.Run("入力は上限の回数まで、使用済みにできるのは一度だけ", func(t *testing.T) {
		codes := repository.NewVerificationCodeRepository(testdb.New(t))
		c := newCode(t0)
		if err := codes.Create(ctx, c); err != nil {
			t.Fatal(err)
		}
		for want := 1; want <= maxAttempts; want++ {
			got, err := codes.ReserveAttempt(ctx, c.ID, maxAttempts, t0)
			if err != nil || got != want {
				t.Fatalf("%d回目: got=%d err=%v", want, got, err)
			}
		}
		if _, err := codes.ReserveAttempt(ctx, c.ID, maxAttempts, t0); !errors.Is(err, repository.ErrConflict) {
			t.Errorf("上限を超えて数えられる: %v", err)
		}
		if _, err := codes.FindLatestActive(ctx, email, domain.VerificationPurposeSignup, maxAttempts, t0); !errors.Is(err, repository.ErrNotFound) {
			t.Errorf("上限に達したコードが見つかる: %v", err)
		}

		c2 := newCode(t0)
		if err := codes.Create(ctx, c2); err != nil {
			t.Fatal(err)
		}
		if err := codes.Consume(ctx, c2.ID, t0); err != nil {
			t.Fatal(err)
		}
		if err := codes.Consume(ctx, c2.ID, t0); !errors.Is(err, repository.ErrConflict) {
			t.Errorf("二度使えてしまう: %v", err)
		}
	})

	t.Run("同時に入力しても上限の回数しか数えない", func(t *testing.T) {
		codes := repository.NewVerificationCodeRepository(testdb.New(t))
		c := newCode(t0)
		if err := codes.Create(ctx, c); err != nil {
			t.Fatal(err)
		}
		var ok atomic.Int32
		var wg sync.WaitGroup
		for range 20 {
			wg.Go(func() {
				if _, err := codes.ReserveAttempt(ctx, c.ID, maxAttempts, t0); err == nil {
					ok.Add(1)
				}
			})
		}
		wg.Wait()
		if ok.Load() != maxAttempts {
			t.Errorf("数えた回数=%d want=%d", ok.Load(), maxAttempts)
		}
	})
}

func newToken(hash string) *domain.PasswordSetupToken {
	return &domain.PasswordSetupToken{
		Email: email, Purpose: domain.PasswordSetupPurposeSignup, TokenHash: hash,
		ExpiresAt: t0.Add(15 * time.Minute), CreatedAt: t0, UpdatedAt: t0,
	}
}

func TestPasswordSetupTokenRepository(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name    string
		hash    string
		purpose domain.PasswordSetupPurpose
		now     time.Time
		ok      bool
	}{
		{"期限内", "h", domain.PasswordSetupPurposeSignup, t0.Add(15*time.Minute - time.Microsecond), true},
		{"15分ちょうどで期限切れ", "h", domain.PasswordSetupPurposeSignup, t0.Add(15 * time.Minute), false},
		{"用途が違う", "h", domain.PasswordSetupPurposePasswordReset, t0, false},
		{"ハッシュが違う", "other", domain.PasswordSetupPurposeSignup, t0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tokens := repository.NewPasswordSetupTokenRepository(testdb.New(t))
			if err := tokens.Create(ctx, newToken("h")); err != nil {
				t.Fatal(err)
			}
			got, err := tokens.Consume(ctx, tc.hash, tc.purpose, tc.now)
			if !tc.ok {
				if !errors.Is(err, repository.ErrConflict) {
					t.Fatalf("ErrConflict にならない: %v", err)
				}
				return
			}
			if err != nil || got.Email != email || got.ConsumedAt == nil {
				t.Fatalf("got=%+v err=%v", got, err)
			}
			if _, err := tokens.Consume(ctx, tc.hash, tc.purpose, tc.now); !errors.Is(err, repository.ErrConflict) {
				t.Errorf("二度使えてしまう: %v", err)
			}
		})
	}

	t.Run("同時に使っても成功するのは1回だけ", func(t *testing.T) {
		tokens := repository.NewPasswordSetupTokenRepository(testdb.New(t))
		if err := tokens.Create(ctx, newToken("race")); err != nil {
			t.Fatal(err)
		}
		var ok atomic.Int32
		var wg sync.WaitGroup
		for range 10 {
			wg.Go(func() {
				if _, err := tokens.Consume(ctx, "race", domain.PasswordSetupPurposeSignup, t0); err == nil {
					ok.Add(1)
				}
			})
		}
		wg.Wait()
		if ok.Load() != 1 {
			t.Errorf("成功した回数=%d", ok.Load())
		}
	})
}

func TestTransactorとセッションと端末(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	tx := repository.NewTransactor(gdb)
	users := repository.NewUserRepository(gdb)
	sessions := repository.NewSessionRepository(gdb)
	devices := repository.NewTrustedDeviceRepository(gdb)

	t.Run("エラーならロールバックする", func(t *testing.T) {
		boom := errors.New("boom")
		err := tx.Do(ctx, func(ctx context.Context) error {
			if err := users.Create(ctx, newUser("rollback", "26238005@edu.cc.saga-u.ac.jp", "26238005")); err != nil {
				return err
			}
			return boom
		})
		if !errors.Is(err, boom) {
			t.Fatalf("err=%v", err)
		}
		if _, err := users.FindByID(ctx, "rollback"); !errors.Is(err, repository.ErrNotFound) {
			t.Errorf("ロールバックされていない: %v", err)
		}
	})

	t.Run("ユーザー・セッション・端末をまとめて作る", func(t *testing.T) {
		u := newUser("", email, "26238001")
		err := tx.Do(ctx, func(ctx context.Context) error {
			if err := users.Create(ctx, u); err != nil {
				return err
			}
			if err := sessions.Create(ctx, &domain.Session{UserID: u.ID, TokenHash: "s", ExpiresAt: t0.Add(time.Hour), CreatedAt: t0, UpdatedAt: t0}); err != nil {
				return err
			}
			return devices.Upsert(ctx, &domain.TrustedDevice{UserID: u.ID, DeviceHash: "d", LastUsedAt: t0, CreatedAt: t0, UpdatedAt: t0})
		})
		if err != nil {
			t.Fatal(err)
		}
		// 同じ端末をもう一度登録すると、行は増えずに最終利用日時が更新される
		later := t0.Add(time.Hour)
		if err := devices.Upsert(ctx, &domain.TrustedDevice{UserID: u.ID, DeviceHash: "d", LastUsedAt: later, CreatedAt: later, UpdatedAt: later}); err != nil {
			t.Fatal(err)
		}
		var n int64
		var lastUsed time.Time
		gdb.Raw("SELECT count(*) FROM trusted_devices WHERE user_id = ?", u.ID).Scan(&n)
		gdb.Raw("SELECT last_used_at FROM trusted_devices WHERE user_id = ?", u.ID).Scan(&lastUsed)
		if n != 1 || !lastUsed.Equal(later) {
			t.Errorf("端末: n=%d last_used_at=%s", n, lastUsed)
		}
		gdb.Raw("SELECT count(*) FROM sessions WHERE user_id = ?", u.ID).Scan(&n)
		if n != 1 {
			t.Errorf("セッション: n=%d", n)
		}
	})
}

func TestSessionRepository_検索と失効(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	users := repository.NewUserRepository(gdb)
	sessions := repository.NewSessionRepository(gdb)
	u := newUser("", email, "26238001")
	if err := users.Create(ctx, u); err != nil {
		t.Fatal(err)
	}
	if err := sessions.Create(ctx, &domain.Session{UserID: u.ID, TokenHash: "s", ExpiresAt: t0.Add(time.Hour), CreatedAt: t0, UpdatedAt: t0}); err != nil {
		t.Fatal(err)
	}

	got, err := sessions.FindActiveByTokenHash(ctx, "s", t0)
	if err != nil || got.UserID != u.ID || !got.ExpiresAt.Equal(t0.Add(time.Hour)) || got.RevokedAt != nil {
		t.Fatalf("期限内: got=%+v err=%v", got, err)
	}
	if _, err := sessions.FindActiveByTokenHash(ctx, "s", t0.Add(time.Hour)); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("期限ちょうどで見つかった: %v", err)
	}
	if _, err := sessions.FindActiveByTokenHash(ctx, "other", t0); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("違うハッシュで見つかった: %v", err)
	}

	if err := sessions.Revoke(ctx, "s", t0); err != nil {
		t.Fatal(err)
	}
	if _, err := sessions.FindActiveByTokenHash(ctx, "s", t0); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("失効後に見つかった: %v", err)
	}
	if err := sessions.Revoke(ctx, "s", t0); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("2回目の失効が ErrNotFound にならない: %v", err)
	}
}

func TestSessionRepository_RevokeAllByUser(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	users := repository.NewUserRepository(gdb)
	sessions := repository.NewSessionRepository(gdb)
	u := newUser("", email, "26238001")
	other := newUser("", "26238002@edu.cc.saga-u.ac.jp", "26238002")
	for _, x := range []*domain.User{u, other} {
		if err := users.Create(ctx, x); err != nil {
			t.Fatal(err)
		}
	}
	for _, s := range []struct{ userID, hash string }{{u.ID, "a"}, {u.ID, "b"}, {other.ID, "c"}} {
		if err := sessions.Create(ctx, &domain.Session{UserID: s.userID, TokenHash: s.hash, ExpiresAt: t0.Add(time.Hour), CreatedAt: t0, UpdatedAt: t0}); err != nil {
			t.Fatal(err)
		}
	}
	// 既に失効したセッションがあっても、まとめて失効できる
	if err := sessions.Revoke(ctx, "b", t0); err != nil {
		t.Fatal(err)
	}

	if err := sessions.RevokeAllByUser(ctx, u.ID, t0); err != nil {
		t.Fatal(err)
	}
	for _, h := range []string{"a", "b"} {
		if _, err := sessions.FindActiveByTokenHash(ctx, h, t0); !errors.Is(err, repository.ErrNotFound) {
			t.Errorf("セッション %s が失効していない: %v", h, err)
		}
	}
	if _, err := sessions.FindActiveByTokenHash(ctx, "c", t0); err != nil {
		t.Errorf("ほかのユーザーのセッションまで失効した: %v", err)
	}
	if err := sessions.RevokeAllByUser(ctx, u.ID, t0); err != nil {
		t.Errorf("対象がないときにエラー: %v", err)
	}
}

func TestTrustedDeviceRepository_Exists(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	users := repository.NewUserRepository(gdb)
	devices := repository.NewTrustedDeviceRepository(gdb)
	u := newUser("", email, "26238001")
	other := newUser("", "26238002@edu.cc.saga-u.ac.jp", "26238002")
	for _, x := range []*domain.User{u, other} {
		if err := users.Create(ctx, x); err != nil {
			t.Fatal(err)
		}
	}
	if err := devices.Upsert(ctx, &domain.TrustedDevice{UserID: u.ID, DeviceHash: "d", LastUsedAt: t0, CreatedAt: t0, UpdatedAt: t0}); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name         string
		userID, hash string
		want         bool
	}{
		{"登録した端末", u.ID, "d", true},
		{"登録していない端末", u.ID, "x", false},
		{"ほかのユーザーが登録した端末", other.ID, "d", false},
	} {
		if got, err := devices.Exists(ctx, tt.userID, tt.hash); err != nil || got != tt.want {
			t.Errorf("%s: got=%v err=%v", tt.name, got, err)
		}
	}
}

func TestLoginChallengeRepository(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	users := repository.NewUserRepository(gdb)
	challenges := repository.NewLoginChallengeRepository(gdb)
	u := newUser("", email, "26238001")
	if err := users.Create(ctx, u); err != nil {
		t.Fatal(err)
	}
	newChallenge := func(hash, device string) *domain.LoginChallenge {
		return &domain.LoginChallenge{
			UserID: u.ID, TokenHash: hash, Reason: domain.LoginRecheckReasonNewDevice, DeviceHash: device,
			ExpiresAt: t0.Add(15 * time.Minute), CreatedAt: t0, UpdatedAt: t0,
		}
	}

	t.Run("保存して読み戻す（端末なしは空）", func(t *testing.T) {
		for _, c := range []*domain.LoginChallenge{newChallenge("with-device", "d"), newChallenge("no-device", "")} {
			if err := challenges.Create(ctx, c); err != nil {
				t.Fatal(err)
			}
			got, err := challenges.FindActive(ctx, c.TokenHash, t0)
			if err != nil {
				t.Fatal(err)
			}
			if got.ID != c.ID || got.UserID != u.ID || got.Reason != domain.LoginRecheckReasonNewDevice ||
				got.DeviceHash != c.DeviceHash || !got.ExpiresAt.Equal(c.ExpiresAt) || got.ConsumedAt != nil {
				t.Errorf("got=%+v want=%+v", got, c)
			}
		}
	})

	t.Run("期限切れは使えない", func(t *testing.T) {
		c := newChallenge("expiring", "")
		if err := challenges.Create(ctx, c); err != nil {
			t.Fatal(err)
		}
		if _, err := challenges.FindActive(ctx, c.TokenHash, c.ExpiresAt); !errors.Is(err, repository.ErrNotFound) {
			t.Errorf("FindActive: %v", err)
		}
		if err := challenges.Consume(ctx, c.ID, c.ExpiresAt); !errors.Is(err, repository.ErrConflict) {
			t.Errorf("Consume: %v", err)
		}
	})

	t.Run("同時に使っても成功するのは1回だけ", func(t *testing.T) {
		c := newChallenge("once", "")
		if err := challenges.Create(ctx, c); err != nil {
			t.Fatal(err)
		}
		var ok atomic.Int32
		var wg sync.WaitGroup
		for range 10 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := challenges.Consume(ctx, c.ID, t0); err == nil {
					ok.Add(1)
				} else if !errors.Is(err, repository.ErrConflict) {
					t.Errorf("想定外のエラー: %v", err)
				}
			}()
		}
		wg.Wait()
		if ok.Load() != 1 {
			t.Errorf("成功した回数=%d want=1", ok.Load())
		}
		if _, err := challenges.FindActive(ctx, c.TokenHash, t0); !errors.Is(err, repository.ErrNotFound) {
			t.Errorf("使用済みが見つかった: %v", err)
		}
	})
}
