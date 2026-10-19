package domain_test

import (
	"testing"

	"github.com/takuma0403/osagari_web/server/internal/domain"
)

func TestParseAdminList(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
		admins  []string
		others  []string
	}{
		{"空なら管理者はいない", "", false, nil, []string{"26141023"}},
		{"空白と空の要素は無視する", " 26141023 ,, 25238117,", false, []string{"26141023", "25238117"}, []string{"24101032"}},
		{"8桁でない", "2614102", true, nil, nil},
		{"数字でない", "26141023,abcdefgh", true, nil, nil},
		{"メールアドレスは受け付けない", "26141023@edu.cc.saga-u.ac.jp", true, nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, err := domain.ParseAdminList(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tt.wantErr)
			}
			for _, n := range tt.admins {
				if !l.Contains(&domain.User{StudentNumber: n}) {
					t.Errorf("%s が管理者になっていない", n)
				}
			}
			for _, n := range tt.others {
				if l.Contains(&domain.User{StudentNumber: n}) {
					t.Errorf("%s が管理者になっている", n)
				}
			}
		})
	}
	if (domain.AdminList{}).Contains(nil) {
		t.Error("nil のユーザーが管理者になっている")
	}
}
