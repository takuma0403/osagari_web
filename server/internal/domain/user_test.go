package domain_test

import (
	"testing"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/domain"
)

// 有効期限は日本時間の日付で判定し、期限の日の終わりまでは使える（AUTH-3）。
func TestUser_IsExpired(t *testing.T) {
	expiresOn := time.Date(2030, 3, 31, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		status domain.UserStatus
		now    time.Time
		want   bool
	}{
		{"期限の前日", domain.UserStatusActive, time.Date(2030, 3, 30, 12, 0, 0, 0, time.UTC), false},
		{"期限の日の終わり（日本時間 3/31 23:59）", domain.UserStatusActive, time.Date(2030, 3, 31, 14, 59, 0, 0, time.UTC), false},
		{"期限の翌日になった瞬間（日本時間 4/1 0:00）", domain.UserStatusActive, time.Date(2030, 3, 31, 15, 0, 0, 0, time.UTC), true},
		{"状態が失効なら期限内でも失効", domain.UserStatusExpired, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), true},
		{"停止は失効ではない", domain.UserStatusSuspended, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := domain.User{Status: tt.status, ExpiresOn: expiresOn}
			if got := u.IsExpired(tt.now); got != tt.want {
				t.Errorf("IsExpired=%v want=%v", got, tt.want)
			}
		})
	}
}
