package domain

import "time"

// StudentCard は学生証（student_cards）。画像そのものは非公開バケットにあり、ここにはオブジェクト名だけを持つ（CARD, STORAGE）。
// 再提出しても前の学生証は SupersededAt を入れて残す（ADR 0014）。
type StudentCard struct {
	ID          string
	UserID      string
	ObjectKey   string
	ContentType string
	SizeBytes   int64
	Status      StudentCardStatus
	SubmittedAt time.Time
	// ReviewedBy・ReviewedAt は審査した管理者と日時（ADMIN）。審査前は nil
	ReviewedBy *string
	ReviewedAt *time.Time
	// SupersededAt は再提出で置き換えた日時。nil なら本人の現在の学生証
	SupersededAt *time.Time
	// DeleteAfter は画像の削除予定日（アカウントの失効・削除日 + 3年）。失効・削除するまでは nil
	DeleteAfter    *time.Time
	ImageDeletedAt *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// StudentCardView は管理者が学生証画像を閲覧した記録（student_card_views。CARD, STORAGE, ADR 0015）。
type StudentCardView struct {
	ID            string
	StudentCardID string
	// AdminUserID は閲覧した管理者のユーザーID
	AdminUserID string
	ViewedAt    time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
