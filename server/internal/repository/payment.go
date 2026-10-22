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
func (r *PaymentRepository) Create(ctx context.Context, p *domain.Payment) error {
	if p.ID == "" {
		p.ID = newID()
	}
	return conn(ctx, r.db).Create(&paymentRow{
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
