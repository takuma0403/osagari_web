package domain_test

import (
	"errors"
	"testing"

	"github.com/takuma0403/osagari_web/server/internal/domain"
)

func TestNormalizeStudentEmail(t *testing.T) {
	cases := []struct {
		name          string
		raw           string
		email, number string
		ok            bool
	}{
		{"学生のアドレス", "26238001@edu.cc.saga-u.ac.jp", "26238001@edu.cc.saga-u.ac.jp", "26238001", true},
		{"前後の空白と大文字は正規化する", "  26238001@EDU.CC.Saga-U.ac.jp\n", "26238001@edu.cc.saga-u.ac.jp", "26238001", true},
		{"7桁", "2623800@edu.cc.saga-u.ac.jp", "", "", false},
		{"9桁", "262380011@edu.cc.saga-u.ac.jp", "", "", false},
		{"数字以外（教職員の形式）", "yamada@edu.cc.saga-u.ac.jp", "", "", false},
		{"全角数字", "２６２３８００１@edu.cc.saga-u.ac.jp", "", "", false},
		{"別のドメイン", "26238001@cc.saga-u.ac.jp", "", "", false},
		{"ドメインの後ろに続きがある", "26238001@edu.cc.saga-u.ac.jp.example.com", "", "", false},
		{"ドットが任意の文字にならない", "26238001@eduxcc.saga-u.ac.jp", "", "", false},
		{"空", "", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			email, number, err := domain.NormalizeStudentEmail(tc.raw)
			if tc.ok != (err == nil) {
				t.Fatalf("err=%v", err)
			}
			if err != nil && !errors.Is(err, domain.ErrInvalidStudentEmail) {
				t.Fatalf("ErrInvalidStudentEmail ではない: %v", err)
			}
			if email != tc.email || number != tc.number {
				t.Errorf("email=%q number=%q", email, number)
			}
		})
	}
}

func TestEstimateStudent(t *testing.T) {
	cases := []struct {
		name       string
		number     string
		year       int
		code       string
		department string // 空なら不明
		expiresOn  string
	}{
		{"理工学部（4年）", "26238001", 2026, "238", "理工学部 理工学科", "2030-03-31"},
		{"医学部医学科（6年）", "26211001", 2026, "211", "医学部 医学科", "2032-03-31"},
		{"医学部看護学科（4年）", "25212010", 2025, "212", "医学部 看護学科", "2029-03-31"},
		{"コスメティックサイエンス学環", "26271001", 2026, "271", "コスメティックサイエンス学環", "2030-03-31"},
		{"大学院（表にない）は不明・4年", "24726001", 2024, "726", "", "2028-03-31"},
		{"改組前のコード（表にない）は不明・4年", "18231001", 2018, "231", "", "2022-03-31"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := domain.EstimateStudent(tc.number)
			if got.StudentNumber != tc.number || got.AdmissionYear != tc.year || got.DepartmentCode != tc.code {
				t.Errorf("got=%+v", got)
			}
			name := ""
			if got.Department != nil {
				name = got.Department.Name
				if got.Department.Code != tc.code {
					t.Errorf("Department.Code=%q", got.Department.Code)
				}
			}
			if name != tc.department {
				t.Errorf("学部=%q want=%q", name, tc.department)
			}
			if d := got.ExpiresOn.Format("2006-01-02"); d != tc.expiresOn {
				t.Errorf("有効期限=%s want=%s", d, tc.expiresOn)
			}
		})
	}
}
