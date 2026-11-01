package domain_test

import (
	"testing"

	"github.com/takuma0403/osagari_web/server/internal/domain"
)

func TestCalcFee(t *testing.T) {
	tests := []struct {
		name         string
		price        int64
		bp           int
		fee, proceed int64
	}{
		{"割り切れる", 1500, 500, 75, 1425},
		{"1円未満は切り捨て", 1550, 500, 77, 1473},
		{"小数点以下2桁の料率", 1000, 525, 52, 948},
		{"料率0%", 1500, 0, 0, 1500},
		{"料率100%", 1500, 10000, 1500, 0},
		{"最低価格", 100, 500, 5, 95},
		{"最高価格", 100000, 999, 9990, 90010},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fee, proceeds := domain.CalcFee(tt.price, tt.bp)
			if fee != tt.fee || proceeds != tt.proceed {
				t.Errorf("CalcFee(%d, %d) = (%d, %d), want (%d, %d)", tt.price, tt.bp, fee, proceeds, tt.fee, tt.proceed)
			}
		})
	}
}
