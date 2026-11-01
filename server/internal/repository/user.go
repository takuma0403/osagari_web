package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/takuma0403/osagari_web/server/internal/domain"
)

// userRow は users の1行（migrations/000001_create_users、000017_add_users_withdrawal_requested_at）。
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
	// WithdrawalRequestedAt は退学を申請した日時（ADR 0034）
	WithdrawalRequestedAt *string
	CreatedAt             string
	UpdatedAt             string
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
	if u.WithdrawalRequestedAt, err = parseTimePtr(r.WithdrawalRequestedAt); err != nil {
		return nil, fmt.Errorf("repository: users.withdrawal_requested_at: %w", err)
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
		ID:                    u.ID,
		Email:                 u.Email,
		StudentNumber:         u.StudentNumber,
		AdmissionYear:         u.AdmissionYear,
		DepartmentCode:        u.DepartmentCode,
		Department:            u.Department,
		ExpiresOn:             u.ExpiresOn.Format(dateLayout),
		Status:                string(u.Status),
		SellerStatus:          string(u.SellerStatus),
		PasswordHash:          u.PasswordHash,
		CodeVerifiedAt:        formatTime(u.CodeVerifiedAt),
		WithdrawalRequestedAt: formatTimePtr(u.WithdrawalRequestedAt),
		CreatedAt:             formatTime(u.CreatedAt),
		UpdatedAt:             formatTime(u.UpdatedAt),
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

// FindByIDs は ID の一覧でユーザーをまとめて探し、ID をキーにして返す。見つからない ID は含めない。
func (r *UserRepository) FindByIDs(ctx context.Context, ids []string) (map[string]*domain.User, error) {
	users := make(map[string]*domain.User, len(ids))
	if len(ids) == 0 {
		return users, nil
	}
	var rows []userRow
	if err := conn(ctx, r.db).Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		u, err := row.toDomain()
		if err != nil {
			return nil, err
		}
		users[u.ID] = u
	}
	return users, nil
}

// FindByEmail はメールアドレスでユーザーを探す。状態にかかわらず返す。なければ ErrNotFound。
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var rows []userRow
	if err := conn(ctx, r.db).Where("email = ?", email).Limit(1).Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return rows[0].toDomain()
}

// UpdateCodeVerifiedAt は最後にメールのコードで確認した日時を更新する（AUTHN-3）。ユーザーがいなければ ErrNotFound。
func (r *UserRepository) UpdateCodeVerifiedAt(ctx context.Context, id string, at time.Time) error {
	ts := formatTime(at)
	res := conn(ctx, r.db).Model(&userRow{}).Where("id = ?", id).
		Updates(map[string]any{"code_verified_at": ts, "updated_at": ts})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdatePassword はパスワードのハッシュを更新する（AUTHN-4）。メールのコードで本人確認をした後に呼ぶので、
// 最後にコードで確認した日時も at にする。ユーザーがいなければ ErrNotFound。
func (r *UserRepository) UpdatePassword(ctx context.Context, id, passwordHash string, at time.Time) error {
	ts := formatTime(at)
	res := conn(ctx, r.db).Model(&userRow{}).Where("id = ?", id).
		Updates(map[string]any{"password_hash": passwordHash, "code_verified_at": ts, "updated_at": ts})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateSellerStatus は出品者の承認状態が from のどれかのときだけ to に更新する（条件付きUPDATE。DB）。
// ユーザーがいない、または状態が from のどれでもないなら ErrConflict。
func (r *UserRepository) UpdateSellerStatus(ctx context.Context, id string, from []domain.SellerStatus, to domain.SellerStatus, at time.Time) error {
	froms := make([]string, len(from))
	for i, s := range from {
		froms[i] = string(s)
	}
	res := conn(ctx, r.db).Model(&userRow{}).Where("id = ? AND seller_status IN ?", id, froms).
		Updates(map[string]any{"seller_status": string(to), "updated_at": formatTime(at)})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrConflict
	}
	return nil
}

// ExistsByEmail はメールアドレスのユーザーがいるかを返す。状態（停止・失効・削除）にかかわらず、行があれば true。
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var n int64
	if err := conn(ctx, r.db).Model(&userRow{}).Where("email = ?", email).Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

// expirableStatuses は有効期限で失効させる状態（有効・停止。ADR 0033）。
var expirableStatuses = []domain.UserStatus{domain.UserStatusActive, domain.UserStatusSuspended}

// ListToExpire は有効期限が today（日本時間の日付）より前で、まだ失効させていないユーザー（有効・停止）を、有効期限の古い順に返す（AUTH-3, JOBS-1）。
func (r *UserRepository) ListToExpire(ctx context.Context, today time.Time) ([]*domain.User, error) {
	var rows []userRow
	err := conn(ctx, r.db).Where("status IN ? AND expires_on < ?", enumStrings(expirableStatuses), today.Format(dateLayout)).
		Order("expires_on ASC, id ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	users := make([]*domain.User, 0, len(rows))
	for _, row := range rows {
		u, err := row.toDomain()
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// Expire は有効・停止で有効期限が today より前のユーザーだけを失効にする（条件付きUPDATE。AUTH-3, DB）。
// ユーザーがいない、またはすでに失効・削除しているなら ErrConflict。
func (r *UserRepository) Expire(ctx context.Context, id string, today, now time.Time) error {
	res := conn(ctx, r.db).Model(&userRow{}).
		Where("id = ? AND status IN ? AND expires_on < ?", id, enumStrings(expirableStatuses), today.Format(dateLayout)).
		Updates(map[string]any{"status": string(domain.UserStatusExpired), "updated_at": formatTime(now)})
	return rowsAffected(res)
}

// withdrawalRequestedCondition は退学を申請して、まだ削除していない人の条件（ADR 0034）。
const withdrawalRequestedCondition = "withdrawal_requested_at IS NOT NULL AND status <> 'deleted'"

// ListForAdmin は管理画面のユーザーを、登録の新しい順（同じなら ID の大きい順）に after より後ろから limit 件返す（ADMIN）。
func (r *UserRepository) ListForAdmin(ctx context.Context, f domain.AdminUserQuery, after *domain.AdminCursor, limit int) ([]*domain.User, error) {
	q := conn(ctx, r.db)
	if f.StudentNumberPrefix != "" {
		// 数字だけなので LIKE のエスケープは要らない（domain.ValidStudentNumberPrefix）
		q = q.Where("student_number LIKE ?", f.StudentNumberPrefix+"%")
	}
	switch f.Filter {
	case domain.AdminUserFilterActive, domain.AdminUserFilterSuspended, domain.AdminUserFilterExpired:
		q = q.Where("status = ?", string(f.Filter))
	case domain.AdminUserFilterWithdrawalRequested:
		q = q.Where(withdrawalRequestedCondition)
	}
	if after != nil {
		at := formatTime(after.At)
		q = q.Where("(created_at < ? OR (created_at = ? AND id < ?))", at, at, after.ID)
	}
	var rows []userRow
	if err := q.Order("created_at DESC, id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	users := make([]*domain.User, 0, len(rows))
	for _, row := range rows {
		u, err := row.toDomain()
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// CountWithdrawalRequested は退学を申請して、まだ削除していない人の数を返す（ADMIN, ADR 0034）。
func (r *UserRepository) CountWithdrawalRequested(ctx context.Context) (int, error) {
	var n int64
	if err := conn(ctx, r.db).Model(&userRow{}).Where(withdrawalRequestedCondition).Count(&n).Error; err != nil {
		return 0, err
	}
	return int(n), nil
}

// UpdateStatus は状態が from のどれかのときだけ to に更新する（条件付きUPDATE。管理者の停止・解除・削除。DB, ADR 0034）。
// ユーザーがいない、または状態が from のどれでもないなら ErrConflict。
func (r *UserRepository) UpdateStatus(ctx context.Context, id string, from []domain.UserStatus, to domain.UserStatus, now time.Time) error {
	res := conn(ctx, r.db).Model(&userRow{}).Where("id = ? AND status IN ?", id, enumStrings(from)).
		Updates(map[string]any{"status": string(to), "updated_at": formatTime(now)})
	return rowsAffected(res)
}

// UpdateExpiresOn は状態が from のどれかのときだけ有効期限を変える（条件付きUPDATE。AUTH-3, ADR 0034）。
// ユーザーがいない、または状態が from のどれでもないなら ErrConflict。
func (r *UserRepository) UpdateExpiresOn(ctx context.Context, id string, from []domain.UserStatus, expiresOn, now time.Time) error {
	res := conn(ctx, r.db).Model(&userRow{}).Where("id = ? AND status IN ?", id, enumStrings(from)).
		Updates(map[string]any{"expires_on": expiresOn.Format(dateLayout), "updated_at": formatTime(now)})
	return rowsAffected(res)
}
