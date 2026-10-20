package repository_test

import (
	"context"
	"testing"

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
