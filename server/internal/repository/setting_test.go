package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/repository"
	"github.com/takuma0403/osagari_web/server/internal/testutil/testdb"
)

func TestSettingRepositoryFeeRate(t *testing.T) {
	settings := repository.NewSettingRepository(testdb.New(t))

	// マイグレーションで入れた暫定の初期値（5%。ADR 0017）
	got, err := settings.FeeRate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got != 500 {
		t.Errorf("FeeRate = %d, want 500", got)
	}
}

// 手数料率を変えると、変更した管理者と日時を記録する（PAY-2, ADR 0034）。
func TestSettingRepositoryUpdateFeeRate(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	settings := repository.NewSettingRepository(gdb)
	admin := newUser("", email, "26238001")
	if err := repository.NewUserRepository(gdb).Create(ctx, admin); err != nil {
		t.Fatal(err)
	}

	s, err := settings.FeeRateSetting(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if s.BasisPoints != 500 || s.UpdatedBy != nil {
		t.Errorf("初期値: %+v", s)
	}

	now := t0.Add(time.Hour)
	if err := settings.UpdateFeeRate(ctx, 1234, admin.ID, now); err != nil {
		t.Fatal(err)
	}
	s, err = settings.FeeRateSetting(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if s.BasisPoints != 1234 || s.UpdatedBy == nil || *s.UpdatedBy != admin.ID || !s.UpdatedAt.Equal(now) {
		t.Errorf("変更後: %+v", s)
	}
	if bp, err := settings.FeeRate(ctx); err != nil || bp != 1234 {
		t.Errorf("FeeRate = %d err=%v", bp, err)
	}
}
