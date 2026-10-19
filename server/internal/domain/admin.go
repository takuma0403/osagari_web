package domain

import (
	"fmt"
	"regexp"
	"strings"
)

var studentNumberPattern = regexp.MustCompile(`^\d{8}$`)

// AdminList は管理者の学籍番号の許可リスト（ADMIN, ADR 0015）。環境変数 ADMIN_STUDENT_NUMBERS から作る。
type AdminList struct {
	numbers map[string]bool
}

// ParseAdminList はカンマ区切りの学籍番号（8桁）を読む。前後の空白と空の要素は無視し、空なら管理者はいない。
// 8桁の数字でない要素があればエラーにする（設定の誤りで管理者が意図せず増減しないよう、起動を止める）。
func ParseAdminList(raw string) (AdminList, error) {
	l := AdminList{numbers: map[string]bool{}}
	for i, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if !studentNumberPattern.MatchString(s) {
			// 値そのものはエラーに入れない（学籍番号をログに出さない）
			return AdminList{}, fmt.Errorf("domain: 管理者の学籍番号の %d 番目が8桁の数字ではありません", i+1)
		}
		l.numbers[s] = true
	}
	return l, nil
}

// Contains はユーザーが管理者かを返す。
func (l AdminList) Contains(u *User) bool {
	return u != nil && l.numbers[u.StudentNumber]
}
