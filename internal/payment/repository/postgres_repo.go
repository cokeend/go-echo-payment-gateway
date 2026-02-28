package repository

import (
	"context"
	"fmt"
	"time"

	"go-payment-gateway/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PostgresPaymentRepository struct {
	db *gorm.DB
}

func NewPostgresPaymentRepository(db *gorm.DB) *PostgresPaymentRepository {
	return &PostgresPaymentRepository{db: db}
}

func (r *PostgresPaymentRepository) Migrate() error {
	return r.db.AutoMigrate(&domain.Payment{})
}

func (r *PostgresPaymentRepository) Create(ctx context.Context, payment *domain.Payment) error {
	if payment.ID == "" {
		payment.ID = uuid.New().String()
	}
	now := time.Now()
	payment.CreatedAt = now
	payment.UpdatedAt = now

	result := r.db.WithContext(ctx).Create(payment)
	if result.Error != nil {
		return fmt.Errorf("insert payment: %w", result.Error)
	}
	return nil
}

func (r *PostgresPaymentRepository) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	var payment domain.Payment
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&payment)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("payment not found: %s", id)
		}
		return nil, fmt.Errorf("get payment by id: %w", result.Error)
	}
	return &payment, nil
}

func (r *PostgresPaymentRepository) GetByStripeID(ctx context.Context, stripeID string) (*domain.Payment, error) {
	var payment domain.Payment
	result := r.db.WithContext(ctx).Where("stripe_payment_id = ?", stripeID).First(&payment)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("payment not found for stripe id: %s", stripeID)
		}
		return nil, fmt.Errorf("get payment by stripe id: %w", result.Error)
	}
	return &payment, nil
}

func (r *PostgresPaymentRepository) UpdateStatus(ctx context.Context, id string, status domain.PaymentStatus, stripePaymentID string) error {
	result := r.db.WithContext(ctx).
		Model(&domain.Payment{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":            status,
			"stripe_payment_id": stripePaymentID,
			"updated_at":        time.Now(),
		})
	if result.Error != nil {
		return fmt.Errorf("update payment status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("payment not found: %s", id)
	}
	return nil
}

func (r *PostgresPaymentRepository) List(ctx context.Context, limit, offset int) ([]*domain.Payment, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	var payments []*domain.Payment
	result := r.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&payments)
	if result.Error != nil {
		return nil, fmt.Errorf("list payments: %w", result.Error)
	}
	return payments, nil
}
