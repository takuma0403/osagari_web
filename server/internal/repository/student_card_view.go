package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/takuma0403/osagari_web/server/internal/domain"
)

// studentCardViewRow は student_card_views の1行（migrations/000009_create_student_card_views）。
type studentCardViewRow struct {
	ID            string
	StudentCardID string
	AdminUserID   string
	ViewedAt      string
	CreatedAt     string
	UpdatedAt     string
}

func (studentCardViewRow) TableName() string { return "student_card_views" }

func (r studentCardViewRow) toDomain() (*domain.StudentCardView, error) {
	v := &domain.StudentCardView{ID: r.ID, StudentCardID: r.StudentCardID, AdminUserID: r.AdminUserID}
	var err error
	for _, f := range []struct {
		dst *time.Time
		src string
	}{{&v.ViewedAt, r.ViewedAt}, {&v.CreatedAt, r.CreatedAt}, {&v.UpdatedAt, r.UpdatedAt}} {
		if *f.dst, err = parseTime(f.src); err != nil {
			return nil, fmt.Errorf("repository: student_card_views: %w", err)
		}
	}
	return v, nil
}

// StudentCardViewRepository は学生証画像の閲覧記録（student_card_views）を読み書きする（CARD, ADR 0015）。記録は消さない。
type StudentCardViewRepository struct {
	db *gorm.DB
}

func NewStudentCardViewRepository(db *gorm.DB) *StudentCardViewRepository {
	return &StudentCardViewRepository{db: db}
}

// Create は閲覧記録を保存する。ID が空なら採番する。
func (r *StudentCardViewRepository) Create(ctx context.Context, v *domain.StudentCardView) error {
	if v.ID == "" {
		v.ID = newID()
	}
	return conn(ctx, r.db).Create(&studentCardViewRow{
		ID:            v.ID,
		StudentCardID: v.StudentCardID,
		AdminUserID:   v.AdminUserID,
		ViewedAt:      formatTime(v.ViewedAt),
		CreatedAt:     formatTime(v.CreatedAt),
		UpdatedAt:     formatTime(v.UpdatedAt),
	}).Error
}

// ListByCard は学生証の閲覧記録を新しい順に返す。
func (r *StudentCardViewRepository) ListByCard(ctx context.Context, studentCardID string) ([]*domain.StudentCardView, error) {
	var rows []studentCardViewRow
	if err := conn(ctx, r.db).Where("student_card_id = ?", studentCardID).
		Order("viewed_at DESC, id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	views := make([]*domain.StudentCardView, 0, len(rows))
	for _, row := range rows {
		v, err := row.toDomain()
		if err != nil {
			return nil, err
		}
		views = append(views, v)
	}
	return views, nil
}
