package domain

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// studentEmailPattern は登録できるメールアドレス（学生の大学メール）。サーバー側で必ず検証する（AUTHN-2）。
// 教職員・留学生の扱いは未決のため、数字8桁の学籍番号だけを受け付ける（AUTH-5）。
var studentEmailPattern = regexp.MustCompile(`^\d{8}@edu\.cc\.saga-u\.ac\.jp$`)

// ErrInvalidStudentEmail はメールアドレスが学生の大学メールの形式ではないときに返す。
var ErrInvalidStudentEmail = errors.New("domain: 大学のメールアドレスの形式ではありません")

// NormalizeStudentEmail は前後の空白を除いて小文字にしたうえで、大学のメールアドレスの形式かを確かめる。
// 正規化したアドレスと、@より前の学籍番号を返す（AUTH-1）。
func NormalizeStudentEmail(raw string) (email, studentNumber string, err error) {
	email = strings.ToLower(strings.TrimSpace(raw))
	if !studentEmailPattern.MatchString(email) {
		return "", "", ErrInvalidStudentEmail
	}
	return email, email[:8], nil
}

// StudentEstimate は学籍番号から推定した情報（AUTH-2〜4）。
type StudentEstimate struct {
	StudentNumber string
	// AdmissionYear は入学年度（上2桁 + 2000）
	AdmissionYear int
	// DepartmentCode は学籍番号の3〜5桁目
	DepartmentCode string
	// Department は AUTH-4 の表で確定した学部・学科。表にないコードは nil（不明）
	Department *Department
	// ExpiresOn は有効期限の初期値（入学年度 + 修業年限 の3月31日、UTC の0時）
	ExpiresOn time.Time
}

// EstimateStudent は8桁の学籍番号から入学年度・学部・有効期限を推定する（AUTH-2〜4）。
// 学籍番号は NormalizeStudentEmail で検証済みのものを渡す。
func EstimateStudent(studentNumber string) StudentEstimate {
	yy, _ := strconv.Atoi(studentNumber[:2])
	code := studentNumber[2:5]
	est := StudentEstimate{
		StudentNumber:  studentNumber,
		AdmissionYear:  2000 + yy,
		DepartmentCode: code,
	}
	years := defaultYears
	if d, ok := LookupDepartment(code); ok {
		est.Department = &d
		years = d.Years
	}
	est.ExpiresOn = time.Date(est.AdmissionYear+years, time.March, 31, 0, 0, 0, 0, time.UTC)
	return est
}
