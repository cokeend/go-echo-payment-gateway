package mocks

import (
	"context"

	"go-payment-gateway/internal/domain"

	"github.com/stretchr/testify/mock"
)

type MockPaymentUseCase struct {
	mock.Mock
}

func (m *MockPaymentUseCase) CreatePayment(ctx context.Context, req *domain.CreatePaymentRequest) (*domain.Payment, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Payment), args.Error(1)
}

func (m *MockPaymentUseCase) CreateCheckout(ctx context.Context, req *domain.CreateCheckoutRequest) (*domain.Payment, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Payment), args.Error(1)
}

func (m *MockPaymentUseCase) GetPayment(ctx context.Context, id string) (*domain.Payment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Payment), args.Error(1)
}

func (m *MockPaymentUseCase) ConfirmPayment(ctx context.Context, id string) (*domain.Payment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Payment), args.Error(1)
}

func (m *MockPaymentUseCase) CancelPayment(ctx context.Context, id string) (*domain.Payment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Payment), args.Error(1)
}

func (m *MockPaymentUseCase) RefundPayment(ctx context.Context, id string, amount int64) (*domain.Payment, error) {
	args := m.Called(ctx, id, amount)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Payment), args.Error(1)
}

func (m *MockPaymentUseCase) HandleWebhook(ctx context.Context, payload []byte, signature string) error {
	args := m.Called(ctx, payload, signature)
	return args.Error(0)
}

func (m *MockPaymentUseCase) ListPayments(ctx context.Context, limit, offset int) ([]*domain.Payment, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Payment), args.Error(1)
}

func (m *MockPaymentUseCase) VerifyWebhook(payload []byte, signature string) (*domain.WebhookEvent, error) {
	args := m.Called(payload, signature)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WebhookEvent), args.Error(1)
}

func (m *MockPaymentUseCase) ProcessWebhookEvent(ctx context.Context, event *domain.WebhookEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}
