package domain_test

import (
	"slices"
	"testing"

	"github.com/takuma0403/osagari_web/server/internal/domain"
)

// 値は docs/glossary.md の「列挙値」表の写し。どちらかを変えたらこのテストが落ちる。
func TestEnums(t *testing.T) {
	tests := []struct {
		name  string
		got   []string
		want  []string
		valid func(string) bool
	}{
		{"UserStatus", strs(domain.UserStatuses), []string{"active", "suspended", "expired", "deleted"},
			func(s string) bool { return domain.UserStatus(s).Valid() }},
		{"SellerStatus", strs(domain.SellerStatuses), []string{"none", "pending", "approved", "rejected"},
			func(s string) bool { return domain.SellerStatus(s).Valid() }},
		{"StudentCardStatus", strs(domain.StudentCardStatuses), []string{"pending", "approved", "rejected"},
			func(s string) bool { return domain.StudentCardStatus(s).Valid() }},
		{"ItemStatus", strs(domain.ItemStatuses), []string{"draft", "published", "requested", "in_deal", "completed", "withdrawn", "suspended"},
			func(s string) bool { return domain.ItemStatus(s).Valid() }},
		{"TransactionStatus", strs(domain.TransactionStatuses), []string{"requested", "rejected", "expired", "in_deal", "cancelled", "completed"},
			func(s string) bool { return domain.TransactionStatus(s).Valid() }},
		{"HandoverMethod", strs(domain.HandoverMethods), []string{"qr", "link", "code", "auto"},
			func(s string) bool { return domain.HandoverMethod(s).Valid() }},
		{"HandoverTokenKind", strs(domain.HandoverTokenKinds), []string{"qr", "link", "code"},
			func(s string) bool { return domain.HandoverTokenKind(s).Valid() }},
		{"PaymentKind", strs(domain.PaymentKinds), []string{"authorize", "capture", "release", "refund", "settle", "payout"},
			func(s string) bool { return domain.PaymentKind(s).Valid() }},
		{"PaymentMethod", strs(domain.PaymentMethods), []string{"card", "paypay"},
			func(s string) bool { return domain.PaymentMethod(s).Valid() }},
		{"PaymentStatus", strs(domain.PaymentStatuses), []string{"pending", "succeeded", "failed"},
			func(s string) bool { return domain.PaymentStatus(s).Valid() }},
		{"ItemCategory", strs(domain.ItemCategories), []string{"textbook", "furniture_appliance", "game_other"},
			func(s string) bool { return domain.ItemCategory(s).Valid() }},
		{"ItemCondition", strs(domain.ItemConditions), []string{"new_like", "good", "fair"},
			func(s string) bool { return domain.ItemCondition(s).Valid() }},
		{"Campus", strs(domain.Campuses), []string{"honjo", "nabeshima"},
			func(s string) bool { return domain.Campus(s).Valid() }},
		{"VerificationPurpose", strs(domain.VerificationPurposes), []string{"signup", "password_reset", "login_recheck"},
			func(s string) bool { return domain.VerificationPurpose(s).Valid() }},
		{"PasswordSetupPurpose", strs(domain.PasswordSetupPurposes), []string{"signup", "password_reset"},
			func(s string) bool { return domain.PasswordSetupPurpose(s).Valid() }},
		{"LoginRecheckReason", strs(domain.LoginRecheckReasons), []string{"code_expired", "new_device"},
			func(s string) bool { return domain.LoginRecheckReason(s).Valid() }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !slices.Equal(tt.got, tt.want) {
				t.Fatalf("値の一覧=%v want=%v", tt.got, tt.want)
			}
			for _, v := range tt.want {
				if !tt.valid(v) {
					t.Errorf("Valid(%q)=false want=true", v)
				}
			}
			for _, v := range []string{"", "unknown", "ACTIVE"} {
				if tt.valid(v) {
					t.Errorf("Valid(%q)=true want=false", v)
				}
			}
		})
	}
}

func strs[T ~string](vs []T) []string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = string(v)
	}
	return out
}
