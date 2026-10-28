package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
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

// 管理画面の一覧の1回に返す件数（ADMIN）。
const (
	AdminListDefaultLimit = 20
	AdminListMaxLimit     = 50
)

// AdminCursor は管理画面の一覧の続きの位置。At の新しい順（同じなら ID の大きい順）で、この行より後ろを返す。
// At はユーザー・商品なら作った日時、取引なら最後に動いた日時。
type AdminCursor struct {
	At time.Time
	ID string
}

// AdminUserFilter は管理画面のユーザーの絞り込み（ADMIN, ADR 0034）。
type AdminUserFilter string

const (
	AdminUserFilterAll       AdminUserFilter = "all"
	AdminUserFilterActive    AdminUserFilter = "active"
	AdminUserFilterSuspended AdminUserFilter = "suspended"
	AdminUserFilterExpired   AdminUserFilter = "expired"
	// AdminUserFilterWithdrawalRequested は退学を申請して、まだ削除していない人
	AdminUserFilterWithdrawalRequested AdminUserFilter = "withdrawal_requested"
)

// AdminUserFilters は AdminUserFilter のすべての値。
var AdminUserFilters = []AdminUserFilter{
	AdminUserFilterAll, AdminUserFilterActive, AdminUserFilterSuspended, AdminUserFilterExpired, AdminUserFilterWithdrawalRequested,
}

func (v AdminUserFilter) Valid() bool { return enumValid(v, AdminUserFilters) }

// AdminUserQuery は管理画面のユーザーの検索条件。
type AdminUserQuery struct {
	// StudentNumberPrefix は学籍番号の前方一致（数字だけ）。空なら絞り込まない
	StudentNumberPrefix string
	Filter              AdminUserFilter
}

var studentNumberPrefixPattern = regexp.MustCompile(`^\d{1,8}$`)

// ValidStudentNumberPrefix は学籍番号の前方一致の検索語（数字1〜8桁）かを返す。
func ValidStudentNumberPrefix(s string) bool { return studentNumberPrefixPattern.MatchString(s) }

// AdminClosableUserStatuses は管理者が削除・有効期限の変更をできるユーザーの状態（有効・停止。ADR 0034）。
var AdminClosableUserStatuses = []UserStatus{UserStatusActive, UserStatusSuspended}
