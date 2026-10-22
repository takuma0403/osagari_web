package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/takuma0403/osagari_web/server/internal/domain"
)

// transactionRow は transactions の1行（migrations/000013_create_transactions）。
type transactionRow struct {
	ID                   string
	ItemID               string
	BuyerID              string
	SellerID             string
	Price                int64
	PaymentMethod        string
	Status               string
	ApprovalDeadline     string
	FeeRateBp            *int
	Fee                  *int64
	SellerProceeds       *int64
	IdentityDisclosedAt  *string
	SellerAcknowledgedAt *string
	BuyerCancelAgreedAt  *string
	SellerCancelAgreedAt *string
	CompletedAt          *string
	HandoverMethod       *string
	CreatedAt            string
	UpdatedAt            string
}

func (transactionRow) TableName() string { return "transactions" }

func (r transactionRow) toDomain() (*domain.Transaction, error) {
	tr := &domain.Transaction{
		ID:             r.ID,
		ItemID:         r.ItemID,
		BuyerID:        r.BuyerID,
		SellerID:       r.SellerID,
		Price:          r.Price,
		PaymentMethod:  domain.PaymentMethod(r.PaymentMethod),
		Status:         domain.TransactionStatus(r.Status),
		FeeRateBP:      r.FeeRateBp,
		Fee:            r.Fee,
		SellerProceeds: r.SellerProceeds,
		HandoverMethod: (*domain.HandoverMethod)(r.HandoverMethod),
	}
	var err error
	for _, f := range []struct {
		name string
		dst  *time.Time
		src  string
	}{
		{"approval_deadline", &tr.ApprovalDeadline, r.ApprovalDeadline},
		{"created_at", &tr.CreatedAt, r.CreatedAt},
		{"updated_at", &tr.UpdatedAt, r.UpdatedAt},
	} {
		if *f.dst, err = parseTime(f.src); err != nil {
			return nil, fmt.Errorf("repository: transactions.%s: %w", f.name, err)
		}
	}
	for _, f := range []struct {
		name string
		dst  **time.Time
		src  *string
	}{
		{"identity_disclosed_at", &tr.IdentityDisclosedAt, r.IdentityDisclosedAt},
		{"seller_acknowledged_at", &tr.SellerAcknowledgedAt, r.SellerAcknowledgedAt},
		{"buyer_cancel_agreed_at", &tr.BuyerCancelAgreedAt, r.BuyerCancelAgreedAt},
		{"seller_cancel_agreed_at", &tr.SellerCancelAgreedAt, r.SellerCancelAgreedAt},
		{"completed_at", &tr.CompletedAt, r.CompletedAt},
	} {
		if *f.dst, err = parseTimePtr(f.src); err != nil {
			return nil, fmt.Errorf("repository: transactions.%s: %w", f.name, err)
		}
	}
	return tr, nil
}

// TransactionRepository は取引（transactions）を読み書きする（DEAL, DATA）。
// 状態遷移（承認・完了・キャンセルなど）は、取引の各機能で条件付きUPDATE のメソッドとして足す（DB）。
type TransactionRepository struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

// Create は取引を保存する。ID が空なら採番する。
// 同じ購入者の同じ商品への購入希望がすでにある、または商品の取引がすでに成立しているときは ErrDuplicate。
func (r *TransactionRepository) Create(ctx context.Context, tr *domain.Transaction) error {
	if tr.ID == "" {
		tr.ID = newID()
	}
	err := conn(ctx, r.db).Create(&transactionRow{
		ID:                   tr.ID,
		ItemID:               tr.ItemID,
		BuyerID:              tr.BuyerID,
		SellerID:             tr.SellerID,
		Price:                tr.Price,
		PaymentMethod:        string(tr.PaymentMethod),
		Status:               string(tr.Status),
		ApprovalDeadline:     formatTime(tr.ApprovalDeadline),
		FeeRateBp:            tr.FeeRateBP,
		Fee:                  tr.Fee,
		SellerProceeds:       tr.SellerProceeds,
		IdentityDisclosedAt:  formatTimePtr(tr.IdentityDisclosedAt),
		SellerAcknowledgedAt: formatTimePtr(tr.SellerAcknowledgedAt),
		BuyerCancelAgreedAt:  formatTimePtr(tr.BuyerCancelAgreedAt),
		SellerCancelAgreedAt: formatTimePtr(tr.SellerCancelAgreedAt),
		CompletedAt:          formatTimePtr(tr.CompletedAt),
		HandoverMethod:       enumPtr(tr.HandoverMethod),
		CreatedAt:            formatTime(tr.CreatedAt),
		UpdatedAt:            formatTime(tr.UpdatedAt),
	}).Error
	if isUniqueViolation(err) {
		return ErrDuplicate
	}
	return err
}

// FindByID は ID で取引を探す。なければ ErrNotFound。
func (r *TransactionRepository) FindByID(ctx context.Context, id string) (*domain.Transaction, error) {
	var rows []transactionRow
	if err := conn(ctx, r.db).Where("id = ?", id).Limit(1).Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return rows[0].toDomain()
}
