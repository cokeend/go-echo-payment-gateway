package mocks

import (
	"context"

	"go-payment-gateway/internal/domain"

	"github.com/stretchr/testify/mock"
)

type MockPaymentRepository struct {
	mock.Mock
}

func (m *MockPaymentRepository) Create(ctx context.Context, payment *domain.Payment) error {
	args := m.Called(ctx, payment)
	return args.Error(0)
}

func (m *MockPaymentRepository) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Payment), args.Error(1)
}

func (m *MockPaymentRepository) GetByStripeID(ctx context.Context, stripeID string) (*domain.Payment, error) {
	args := m.Called(ctx, stripeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Payment), args.Error(1)
}

func (m *MockPaymentRepository) UpdateStatus(ctx context.Context, id string, status domain.PaymentStatus, stripePaymentID string) error {
	args := m.Called(ctx, id, status, stripePaymentID)
	return args.Error(0)
}

func (m *MockPaymentRepository) List(ctx context.Context, limit, offset int) ([]*domain.Payment, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Payment), args.Error(1)
}
