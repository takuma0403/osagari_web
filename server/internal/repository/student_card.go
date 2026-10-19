package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/takuma0403/osagari_web/server/internal/domain"
)

// studentCardRow は student_cards の1行（migrations/000008_create_student_cards）。
type studentCardRow struct {
	ID             string
	UserID         string
	ObjectKey      string
	ContentType    string
	SizeBytes      int64
	Status         string
	SubmittedAt    string
	ReviewedBy     *string
	ReviewedAt     *string
	SupersededAt   *string
	DeleteAfter    *string
	ImageDeletedAt *string
	CreatedAt      string
	UpdatedAt      string
}

func (studentCardRow) TableName() string { return "student_cards" }

func (r studentCardRow) toDomain() (*domain.StudentCard, error) {
	c := &domain.StudentCard{
		ID:          r.ID,
		UserID:      r.UserID,
		ObjectKey:   r.ObjectKey,
		ContentType: r.ContentType,
		SizeBytes:   r.SizeBytes,
		Status:      domain.StudentCardStatus(r.Status),
		ReviewedBy:  r.ReviewedBy,
	}
	var err error
	for _, f := range []struct {
		dst *time.Time
		src string
	}{{&c.SubmittedAt, r.SubmittedAt}, {&c.CreatedAt, r.CreatedAt}, {&c.UpdatedAt, r.UpdatedAt}} {
		if *f.dst, err = parseTime(f.src); err != nil {
			return nil, fmt.Errorf("repository: student_cards: %w", err)
		}
	}
	for _, f := range []struct {
		dst **time.Time
		src *string
	}{{&c.ReviewedAt, r.ReviewedAt}, {&c.SupersededAt, r.SupersededAt}, {&c.ImageDeletedAt, r.ImageDeletedAt}} {
		if *f.dst, err = parseTimePtr(f.src); err != nil {
			return nil, fmt.Errorf("repository: student_cards: %w", err)
		}
	}
	if r.DeleteAfter != nil {
		d, err := parseDate(*r.DeleteAfter)
		if err != nil {
			return nil, fmt.Errorf("repository: student_cards.delete_after: %w", err)
		}
		c.DeleteAfter = &d
	}
	return c, nil
}

// StudentCardRepository は student_cards を読み書きする（CARD）。
type StudentCardRepository struct {
	db *gorm.DB
}

func NewStudentCardRepository(db *gorm.DB) *StudentCardRepository {
	return &StudentCardRepository{db: db}
}

// Create は学生証を保存する。ID が空なら採番する。
// 同じオブジェクト名がすでにある、または本人の現在の学生証（superseded_at が NULL）がすでにあるなら ErrDuplicate。
func (r *StudentCardRepository) Create(ctx context.Context, c *domain.StudentCard) error {
	if c.ID == "" {
		c.ID = newID()
	}
	var deleteAfter *string
	if c.DeleteAfter != nil {
		d := c.DeleteAfter.UTC().Format(dateLayout)
		deleteAfter = &d
	}
	row := studentCardRow{
		ID:             c.ID,
		UserID:         c.UserID,
		ObjectKey:      c.ObjectKey,
		ContentType:    c.ContentType,
		SizeBytes:      c.SizeBytes,
		Status:         string(c.Status),
		SubmittedAt:    formatTime(c.SubmittedAt),
		ReviewedBy:     c.ReviewedBy,
		ReviewedAt:     formatTimePtr(c.ReviewedAt),
		SupersededAt:   formatTimePtr(c.SupersededAt),
		DeleteAfter:    deleteAfter,
		ImageDeletedAt: formatTimePtr(c.ImageDeletedAt),
		CreatedAt:      formatTime(c.CreatedAt),
		UpdatedAt:      formatTime(c.UpdatedAt),
	}
	err := conn(ctx, r.db).Create(&row).Error
	if isUniqueViolation(err) {
		return ErrDuplicate
	}
	return err
}

// FindCurrentByUser は本人の現在の学生証（置き換えられていないもの）を返す。なければ ErrNotFound。
func (r *StudentCardRepository) FindCurrentByUser(ctx context.Context, userID string) (*domain.StudentCard, error) {
	var rows []studentCardRow
	if err := conn(ctx, r.db).Where("user_id = ? AND superseded_at IS NULL", userID).Limit(1).Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return rows[0].toDomain()
}

// ExistsByObjectKey はオブジェクト名がすでに登録されているかを返す。置き換えた学生証も含む。
func (r *StudentCardRepository) ExistsByObjectKey(ctx context.Context, objectKey string) (bool, error) {
	var n int64
	if err := conn(ctx, r.db).Model(&studentCardRow{}).Where("object_key = ?", objectKey).Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

// SupersedeCurrent は本人の現在の学生証が審査中か差し戻しなら、置き換えた日時を入れる（再提出。ADR 0014）。
// 行は消さない。現在の学生証がない、またはすでに承認済みで置き換えなかったときは false を返す（条件付きUPDATE。DB）。
func (r *StudentCardRepository) SupersedeCurrent(ctx context.Context, userID string, now time.Time) (bool, error) {
	ts := formatTime(now)
	res := conn(ctx, r.db).Model(&studentCardRow{}).
		Where("user_id = ? AND superseded_at IS NULL AND status IN ?", userID,
			[]string{string(domain.StudentCardStatusPending), string(domain.StudentCardStatusRejected)}).
		Updates(map[string]any{"superseded_at": ts, "updated_at": ts})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// FindByID は ID で学生証を探す。置き換えた学生証も返す。なければ ErrNotFound。
func (r *StudentCardRepository) FindByID(ctx context.Context, id string) (*domain.StudentCard, error) {
	var rows []studentCardRow
	if err := conn(ctx, r.db).Where("id = ?", id).Limit(1).Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return rows[0].toDomain()
}

// ListPending は審査待ちで置き換えられていない学生証を、提出の古い順に返す（ADMIN）。
func (r *StudentCardRepository) ListPending(ctx context.Context) ([]*domain.StudentCard, error) {
	var rows []studentCardRow
	if err := conn(ctx, r.db).
		Where("status = ? AND superseded_at IS NULL", string(domain.StudentCardStatusPending)).
		Order("submitted_at ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	cards := make([]*domain.StudentCard, 0, len(rows))
	for _, row := range rows {
		c, err := row.toDomain()
		if err != nil {
			return nil, err
		}
		cards = append(cards, c)
	}
	return cards, nil
}

// Review は審査待ちで置き換えられていない学生証だけを to（承認・差し戻し）にし、審査した管理者と日時を入れる（条件付きUPDATE。DB）。
// 学生証がない、または審査待ちでない・置き換えられたなら ErrConflict。
func (r *StudentCardRepository) Review(ctx context.Context, id string, to domain.StudentCardStatus, adminUserID string, now time.Time) error {
	ts := formatTime(now)
	res := conn(ctx, r.db).Model(&studentCardRow{}).
		Where("id = ? AND status = ? AND superseded_at IS NULL", id, string(domain.StudentCardStatusPending)).
		Updates(map[string]any{"status": string(to), "reviewed_by": adminUserID, "reviewed_at": ts, "updated_at": ts})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrConflict
	}
	return nil
}
