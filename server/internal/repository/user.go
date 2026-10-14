package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/takuma0403/osagari_web/server/internal/domain"
)

// userRow は users の1行（migrations/000001_create_users）。
type userRow struct {
	ID             string
	Email          string
	StudentNumber  string
	AdmissionYear  int
	DepartmentCode string
	Department     *string
	ExpiresOn      string
	Status         string
	SellerStatus   string
	PasswordHash   string
	CodeVerifiedAt string
	CreatedAt      string
	UpdatedAt      string
}

func (userRow) TableName() string { return "users" }

func (r userRow) toDomain() (*domain.User, error) {
	expiresOn, err := parseDate(r.ExpiresOn)
	if err != nil {
		return nil, fmt.Errorf("repository: users.expires_on: %w", err)
	}
	u := &domain.User{
		ID:             r.ID,
		Email:          r.Email,
		StudentNumber:  r.StudentNumber,
		AdmissionYear:  r.AdmissionYear,
		DepartmentCode: r.DepartmentCode,
		Department:     r.Department,
		ExpiresOn:      expiresOn,
		Status:         domain.UserStatus(r.Status),
		SellerStatus:   domain.SellerStatus(r.SellerStatus),
		PasswordHash:   r.PasswordHash,
	}
	for _, f := range []struct {
		dst *time.Time
		src string
	}{{&u.CodeVerifiedAt, r.CodeVerifiedAt}, {&u.CreatedAt, r.CreatedAt}, {&u.UpdatedAt, r.UpdatedAt}} {
		if *f.dst, err = parseTime(f.src); err != nil {
			return nil, fmt.Errorf("repository: users: %w", err)
		}
	}
	return u, nil
}

// UserRepository は users を読み書きする。
type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create はユーザーを作る。ID が空なら採番する。メールアドレスか学籍番号が既にあれば ErrDuplicate。
func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	if u.ID == "" {
		u.ID = newID()
	}
	row := userRow{
		ID:             u.ID,
		Email:          u.Email,
		StudentNumber:  u.StudentNumber,
		AdmissionYear:  u.AdmissionYear,
		DepartmentCode: u.DepartmentCode,
		Department:     u.Department,
		ExpiresOn:      u.ExpiresOn.Format(dateLayout),
		Status:         string(u.Status),
		SellerStatus:   string(u.SellerStatus),
		PasswordHash:   u.PasswordHash,
		CodeVerifiedAt: formatTime(u.CodeVerifiedAt),
		CreatedAt:      formatTime(u.CreatedAt),
		UpdatedAt:      formatTime(u.UpdatedAt),
	}
	if err := conn(ctx, r.db).Create(&row).Error; err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicate
		}
		return err
	}
	return nil
}

// FindByID は ID でユーザーを探す。なければ ErrNotFound。
func (r *UserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	var rows []userRow
	if err := conn(ctx, r.db).Where("id = ?", id).Limit(1).Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return rows[0].toDomain()
}

// ExistsByEmail はメールアドレスのユーザーがいるかを返す。状態（停止・失効・削除）にかかわらず、行があれば true。
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var n int64
	if err := conn(ctx, r.db).Model(&userRow{}).Where("email = ?", email).Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}
