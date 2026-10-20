package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/service"
)

type fakeSettingStore struct {
	bp  int
	err error
}

func (f fakeSettingStore) FeeRate(context.Context) (int, error) { return f.bp, f.err }

func TestSettingServiceFeeRate(t *testing.T) {
	ctx := context.Background()

	got, err := service.NewSettingService(service.SettingDeps{Settings: fakeSettingStore{bp: 500}}).FeeRate(ctx)
	if err != nil || got != 500 {
		t.Fatalf("FeeRate = (%d, %v), want (500, nil)", got, err)
	}

	// 読み出せないときは内部エラーにする（DB のエラーを応答に出さない）
	_, err = service.NewSettingService(service.SettingDeps{Settings: fakeSettingStore{err: errors.New("db")}}).FeeRate(ctx)
	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != apperr.CodeInternal {
		t.Fatalf("err = %v, want internal", err)
	}
}
