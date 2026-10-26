package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/takuma0403/osagari_web/server/internal/domain"
)

// paymentRow は payments の1行（migrations/000014_create_payments）。
type paymentRow struct {
	ID            string
	TransactionID string
	Kind          string
	PaymentMethod *string
	Amount        int64
	Status        string
	ExternalID    *string
	CreatedAt     string
	UpdatedAt     string
}

func (paymentRow) TableName() string { return "payments" }

func (r paymentRow) toDomain() (*domain.Payment, error) {
	p := &domain.Payment{
		ID:            r.ID,
		TransactionID: r.TransactionID,
		Kind:          domain.PaymentKind(r.Kind),
		PaymentMethod: (*domain.PaymentMethod)(r.PaymentMethod),
		Amount:        r.Amount,
		Status:        domain.PaymentStatus(r.Status),
		ExternalID:    r.ExternalID,
	}
	var err error
	if p.CreatedAt, err = parseTime(r.CreatedAt); err != nil {
		return nil, fmt.Errorf("repository: payments.created_at: %w", err)
	}
	if p.UpdatedAt, err = parseTime(r.UpdatedAt); err != nil {
		return nil, fmt.Errorf("repository: payments.updated_at: %w", err)
	}
	return p, nil
}

// PaymentRepository は決済・出金の記録（payments）を読み書きする（PAY, JOBS-2）。
type PaymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

// Create は決済の記録を保存する。ID が空なら採番する。
// 同じ取引の出金（payout）がすでにあれば ErrDuplicate（ADR 0029）。
func (r *PaymentRepository) Create(ctx context.Context, p *domain.Payment) error {
	if p.ID == "" {
		p.ID = newID()
	}
	err := conn(ctx, r.db).Create(&paymentRow{
		ID:            p.ID,
		TransactionID: p.TransactionID,
		Kind:          string(p.Kind),
		PaymentMethod: enumPtr(p.PaymentMethod),
		Amount:        p.Amount,
		Status:        string(p.Status),
		ExternalID:    p.ExternalID,
		CreatedAt:     formatTime(p.CreatedAt),
		UpdatedAt:     formatTime(p.UpdatedAt),
	}).Error
	if isUniqueViolation(err) {
		return ErrDuplicate
	}
	return err
}

// ListPayoutsByTransactions は取引ごとの出金（payout）の記録を、取引IDごとの対応で返す（売上の明細。ADR 0029）。
// 出金していない取引は結果に含めない。
func (r *PaymentRepository) ListPayoutsByTransactions(ctx context.Context, transactionIDs []string) (map[string]*domain.Payment, error) {
	out := make(map[string]*domain.Payment, len(transactionIDs))
	if len(transactionIDs) == 0 {
		return out, nil
	}
	var rows []paymentRow
	if err := conn(ctx, r.db).Where("transaction_id IN ? AND kind = ?", transactionIDs, string(domain.PaymentKindPayout)).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		p, err := row.toDomain()
		if err != nil {
			return nil, err
		}
		out[p.TransactionID] = p
	}
	return out, nil
}

// ListByTransaction は取引の決済の記録を古い順に返す。
func (r *PaymentRepository) ListByTransaction(ctx context.Context, transactionID string) ([]*domain.Payment, error) {
	var rows []paymentRow
	if err := conn(ctx, r.db).Where("transaction_id = ?", transactionID).
		Order("created_at, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	payments := make([]*domain.Payment, 0, len(rows))
	for _, row := range rows {
		p, err := row.toDomain()
		if err != nil {
			return nil, err
		}
		payments = append(payments, p)
	}
	return payments, nil
}
