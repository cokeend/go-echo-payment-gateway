package mocks

import (
	"context"

	"go-payment-gateway/internal/domain"

	"github.com/stretchr/testify/mock"
)

type MockPaymentGateway struct {
	mock.Mock
}

func (m *MockPaymentGateway) CreatePaymentIntent(ctx context.Context, amount int64, currency string, paymentMethods []domain.PaymentMethodType, metadata map[string]string) (*domain.PaymentIntentResult, error) {
	args := m.Called(ctx, amount, currency, paymentMethods, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PaymentIntentResult), args.Error(1)
}

func (m *MockPaymentGateway) CreateCheckoutSession(ctx context.Context, req *domain.CreateCheckoutRequest, metadata map[string]string) (*domain.CheckoutSessionResult, error) {
	args := m.Called(ctx, req, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CheckoutSessionResult), args.Error(1)
}

func (m *MockPaymentGateway) ConfirmPaymentIntent(ctx context.Context, paymentIntentID string) (*domain.PaymentIntentResult, error) {
	args := m.Called(ctx, paymentIntentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PaymentIntentResult), args.Error(1)
}

func (m *MockPaymentGateway) CancelPaymentIntent(ctx context.Context, paymentIntentID string) error {
	args := m.Called(ctx, paymentIntentID)
	return args.Error(0)
}

func (m *MockPaymentGateway) CreateRefund(ctx context.Context, paymentIntentID string, amount int64) (*domain.RefundResult, error) {
	args := m.Called(ctx, paymentIntentID, amount)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.RefundResult), args.Error(1)
}

func (m *MockPaymentGateway) ConstructWebhookEvent(payload []byte, signature string) (*domain.WebhookEvent, error) {
	args := m.Called(payload, signature)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WebhookEvent), args.Error(1)
}
