package usecase

import (
	"context"
	"errors"
	"testing"

	"go-payment-gateway/internal/domain"
	"go-payment-gateway/internal/domain/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestUseCase() (*paymentUseCase, *mocks.MockPaymentRepository, *mocks.MockPaymentGateway) {
	repo := new(mocks.MockPaymentRepository)
	gw := new(mocks.MockPaymentGateway)
	uc := &paymentUseCase{repo: repo, gateway: gw}
	return uc, repo, gw
}

// --- CreatePayment ---

func TestCreatePayment_Success(t *testing.T) {
	uc, repo, gw := newTestUseCase()
	ctx := context.Background()

	gw.On("CreatePaymentIntent", ctx, int64(1000), "thb", []domain.PaymentMethodType(nil), mock.Anything).
		Return(&domain.PaymentIntentResult{ID: "pi_123", ClientSecret: "secret_123", Status: "requires_payment_method"}, nil)

	repo.On("Create", ctx, mock.AnythingOfType("*domain.Payment")).
		Return(nil)

	payment, err := uc.CreatePayment(ctx, &domain.CreatePaymentRequest{
		Amount:        1000,
		Currency:      "thb",
		CustomerEmail: "test@example.com",
		Description:   "Test payment",
	})

	assert.NoError(t, err)
	assert.Equal(t, domain.StatusPending, payment.Status)
	assert.Equal(t, "pi_123", payment.StripePaymentID)
	assert.Equal(t, "secret_123", payment.ClientSecret)
	assert.Equal(t, int64(1000), payment.Amount)
	assert.Equal(t, "thb", payment.Currency)
	gw.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestCreatePayment_UnsupportedCurrency(t *testing.T) {
	uc, _, _ := newTestUseCase()

	_, err := uc.CreatePayment(context.Background(), &domain.CreatePaymentRequest{
		Amount:        1000,
		Currency:      "xyz",
		CustomerEmail: "test@example.com",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported currency")
}

func TestCreatePayment_InvalidPaymentMethod(t *testing.T) {
	uc, _, _ := newTestUseCase()

	_, err := uc.CreatePayment(context.Background(), &domain.CreatePaymentRequest{
		Amount:         1000,
		Currency:       "thb",
		CustomerEmail:  "test@example.com",
		PaymentMethods: []domain.PaymentMethodType{"invalid_method"},
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported payment method")
}

func TestCreatePayment_MethodCurrencyMismatch(t *testing.T) {
	uc, _, _ := newTestUseCase()

	_, err := uc.CreatePayment(context.Background(), &domain.CreatePaymentRequest{
		Amount:         1000,
		Currency:       "usd",
		CustomerEmail:  "test@example.com",
		PaymentMethods: []domain.PaymentMethodType{domain.MethodPromptPay},
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires currency THB")
}

func TestCreatePayment_GatewayError(t *testing.T) {
	uc, _, gw := newTestUseCase()
	ctx := context.Background()

	gw.On("CreatePaymentIntent", ctx, int64(1000), "thb", []domain.PaymentMethodType(nil), mock.Anything).
		Return(nil, errors.New("stripe error"))

	_, err := uc.CreatePayment(ctx, &domain.CreatePaymentRequest{
		Amount:        1000,
		Currency:      "thb",
		CustomerEmail: "test@example.com",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create payment intent")
}

func TestCreatePayment_RepoError(t *testing.T) {
	uc, repo, gw := newTestUseCase()
	ctx := context.Background()

	gw.On("CreatePaymentIntent", ctx, int64(500), "usd", []domain.PaymentMethodType(nil), mock.Anything).
		Return(&domain.PaymentIntentResult{ID: "pi_456", ClientSecret: "s_456"}, nil)

	repo.On("Create", ctx, mock.AnythingOfType("*domain.Payment")).
		Return(errors.New("db error"))

	_, err := uc.CreatePayment(ctx, &domain.CreatePaymentRequest{
		Amount:        500,
		Currency:      "usd",
		CustomerEmail: "test@example.com",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save payment")
}

// --- CreateCheckout ---

func TestCreateCheckout_Success(t *testing.T) {
	uc, repo, gw := newTestUseCase()
	ctx := context.Background()

	gw.On("CreateCheckoutSession", ctx, mock.AnythingOfType("*domain.CreateCheckoutRequest"), mock.Anything).
		Return(&domain.CheckoutSessionResult{
			SessionID:   "cs_123",
			CheckoutURL: "https://checkout.stripe.com/cs_123",
		}, nil)

	repo.On("Create", ctx, mock.AnythingOfType("*domain.Payment")).
		Return(nil)

	payment, err := uc.CreateCheckout(ctx, &domain.CreateCheckoutRequest{
		Amount:        2000,
		Currency:      "THB",
		CustomerEmail: "test@example.com",
		SuccessURL:    "https://example.com/success",
		CancelURL:     "https://example.com/cancel",
	})

	assert.NoError(t, err)
	assert.Equal(t, domain.StatusPending, payment.Status)
	assert.Equal(t, "cs_123", payment.StripePaymentID)
	assert.Equal(t, "https://checkout.stripe.com/cs_123", payment.CheckoutURL)
	assert.Equal(t, "thb", payment.Currency)
}

func TestCreateCheckout_WithPaymentIntentID(t *testing.T) {
	uc, repo, gw := newTestUseCase()
	ctx := context.Background()

	gw.On("CreateCheckoutSession", ctx, mock.Anything, mock.Anything).
		Return(&domain.CheckoutSessionResult{
			SessionID:       "cs_456",
			CheckoutURL:     "https://checkout.stripe.com/cs_456",
			PaymentIntentID: "pi_789",
		}, nil)

	repo.On("Create", ctx, mock.AnythingOfType("*domain.Payment")).
		Return(nil)

	payment, err := uc.CreateCheckout(ctx, &domain.CreateCheckoutRequest{
		Amount:        1000,
		Currency:      "usd",
		CustomerEmail: "test@example.com",
		SuccessURL:    "https://example.com/success",
		CancelURL:     "https://example.com/cancel",
	})

	assert.NoError(t, err)
	assert.Equal(t, "pi_789", payment.StripePaymentID)
}

// --- GetPayment ---

func TestGetPayment_Success(t *testing.T) {
	uc, repo, _ := newTestUseCase()
	ctx := context.Background()

	expected := &domain.Payment{ID: "pay_1", Amount: 1000, Status: domain.StatusPending}
	repo.On("GetByID", ctx, "pay_1").Return(expected, nil)

	payment, err := uc.GetPayment(ctx, "pay_1")

	assert.NoError(t, err)
	assert.Equal(t, expected, payment)
}

func TestGetPayment_NotFound(t *testing.T) {
	uc, repo, _ := newTestUseCase()
	ctx := context.Background()

	repo.On("GetByID", ctx, "not_exist").Return(nil, errors.New("payment not found"))

	_, err := uc.GetPayment(ctx, "not_exist")

	assert.Error(t, err)
}

// --- ConfirmPayment ---

func TestConfirmPayment_Success(t *testing.T) {
	uc, repo, gw := newTestUseCase()
	ctx := context.Background()

	payment := &domain.Payment{ID: "pay_1", StripePaymentID: "pi_123", Status: domain.StatusPending}
	repo.On("GetByID", ctx, "pay_1").Return(payment, nil)
	gw.On("ConfirmPaymentIntent", ctx, "pi_123").
		Return(&domain.PaymentIntentResult{ID: "pi_123", Status: "succeeded"}, nil)
	repo.On("UpdateStatus", ctx, "pay_1", domain.StatusSucceeded, "pi_123").Return(nil)

	result, err := uc.ConfirmPayment(ctx, "pay_1")

	assert.NoError(t, err)
	assert.Equal(t, domain.StatusSucceeded, result.Status)
}

func TestConfirmPayment_NotPending(t *testing.T) {
	uc, repo, _ := newTestUseCase()
	ctx := context.Background()

	payment := &domain.Payment{ID: "pay_1", Status: domain.StatusSucceeded}
	repo.On("GetByID", ctx, "pay_1").Return(payment, nil)

	_, err := uc.ConfirmPayment(ctx, "pay_1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in pending state")
}

// --- CancelPayment ---

func TestCancelPayment_Success(t *testing.T) {
	uc, repo, gw := newTestUseCase()
	ctx := context.Background()

	payment := &domain.Payment{ID: "pay_1", StripePaymentID: "pi_123", Status: domain.StatusPending}
	repo.On("GetByID", ctx, "pay_1").Return(payment, nil)
	gw.On("CancelPaymentIntent", ctx, "pi_123").Return(nil)
	repo.On("UpdateStatus", ctx, "pay_1", domain.StatusCanceled, "pi_123").Return(nil)

	result, err := uc.CancelPayment(ctx, "pay_1")

	assert.NoError(t, err)
	assert.Equal(t, domain.StatusCanceled, result.Status)
}

func TestCancelPayment_NotPending(t *testing.T) {
	uc, repo, _ := newTestUseCase()
	ctx := context.Background()

	payment := &domain.Payment{ID: "pay_1", Status: domain.StatusSucceeded}
	repo.On("GetByID", ctx, "pay_1").Return(payment, nil)

	_, err := uc.CancelPayment(ctx, "pay_1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only pending payments")
}

// --- RefundPayment ---

func TestRefundPayment_Success(t *testing.T) {
	uc, repo, gw := newTestUseCase()
	ctx := context.Background()

	payment := &domain.Payment{ID: "pay_1", StripePaymentID: "pi_123", Amount: 1000, Status: domain.StatusSucceeded}
	repo.On("GetByID", ctx, "pay_1").Return(payment, nil)
	gw.On("CreateRefund", ctx, "pi_123", int64(500)).
		Return(&domain.RefundResult{ID: "re_1", Status: "succeeded", Amount: 500}, nil)
	repo.On("UpdateStatus", ctx, "pay_1", domain.StatusRefunded, "pi_123").Return(nil)

	result, err := uc.RefundPayment(ctx, "pay_1", 500)

	assert.NoError(t, err)
	assert.Equal(t, domain.StatusRefunded, result.Status)
}

func TestRefundPayment_NotSucceeded(t *testing.T) {
	uc, repo, _ := newTestUseCase()
	ctx := context.Background()

	payment := &domain.Payment{ID: "pay_1", Status: domain.StatusPending}
	repo.On("GetByID", ctx, "pay_1").Return(payment, nil)

	_, err := uc.RefundPayment(ctx, "pay_1", 500)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only succeeded payments")
}

func TestRefundPayment_ExceedsAmount(t *testing.T) {
	uc, repo, _ := newTestUseCase()
	ctx := context.Background()

	payment := &domain.Payment{ID: "pay_1", Amount: 1000, Status: domain.StatusSucceeded}
	repo.On("GetByID", ctx, "pay_1").Return(payment, nil)

	_, err := uc.RefundPayment(ctx, "pay_1", 2000)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds payment amount")
}

// --- HandleWebhook ---

func TestHandleWebhook_CheckoutSessionCompleted(t *testing.T) {
	uc, repo, gw := newTestUseCase()
	ctx := context.Background()

	event := &domain.WebhookEvent{
		Type:            "checkout.session.completed",
		SessionID:       "cs_123",
		PaymentIntentID: "pi_456",
		Status:          "complete",
	}

	gw.On("ConstructWebhookEvent", []byte("payload"), "sig").Return(event, nil)

	payment := &domain.Payment{ID: "pay_1", StripePaymentID: "cs_123"}
	repo.On("GetByStripeID", ctx, "cs_123").Return(payment, nil)
	repo.On("UpdateStatus", ctx, "pay_1", domain.StatusSucceeded, "pi_456").Return(nil)

	err := uc.HandleWebhook(ctx, []byte("payload"), "sig")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestHandleWebhook_PaymentIntentSucceeded(t *testing.T) {
	uc, repo, gw := newTestUseCase()
	ctx := context.Background()

	event := &domain.WebhookEvent{
		Type:            "payment_intent.succeeded",
		PaymentIntentID: "pi_123",
		Status:          "succeeded",
	}

	gw.On("ConstructWebhookEvent", []byte("payload"), "sig").Return(event, nil)

	payment := &domain.Payment{ID: "pay_1", StripePaymentID: "pi_123", Status: domain.StatusPending}
	repo.On("GetByStripeID", ctx, "pi_123").Return(payment, nil)
	repo.On("UpdateStatus", ctx, "pay_1", domain.StatusSucceeded, "pi_123").Return(nil)

	err := uc.HandleWebhook(ctx, []byte("payload"), "sig")

	assert.NoError(t, err)
}

func TestHandleWebhook_InvalidSignature(t *testing.T) {
	uc, _, gw := newTestUseCase()
	ctx := context.Background()

	gw.On("ConstructWebhookEvent", []byte("payload"), "bad_sig").
		Return(nil, errors.New("invalid signature"))

	err := uc.HandleWebhook(ctx, []byte("payload"), "bad_sig")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "construct webhook event")
}

func TestHandleWebhook_UnknownEventIgnored(t *testing.T) {
	uc, _, gw := newTestUseCase()
	ctx := context.Background()

	event := &domain.WebhookEvent{Type: "customer.created"}
	gw.On("ConstructWebhookEvent", []byte("payload"), "sig").Return(event, nil)

	err := uc.HandleWebhook(ctx, []byte("payload"), "sig")

	assert.NoError(t, err)
}

// --- VerifyWebhook ---

func TestVerifyWebhook_Success(t *testing.T) {
	uc, _, gw := newTestUseCase()

	expected := &domain.WebhookEvent{Type: "payment_intent.succeeded"}
	gw.On("ConstructWebhookEvent", []byte("payload"), "sig").Return(expected, nil)

	event, err := uc.VerifyWebhook([]byte("payload"), "sig")

	assert.NoError(t, err)
	assert.Equal(t, expected, event)
}

// --- ProcessWebhookEvent ---

func TestProcessWebhookEvent_CheckoutCompleted(t *testing.T) {
	uc, repo, _ := newTestUseCase()
	ctx := context.Background()

	event := &domain.WebhookEvent{
		Type:            "checkout.session.completed",
		SessionID:       "cs_100",
		PaymentIntentID: "pi_200",
	}

	payment := &domain.Payment{ID: "pay_1", StripePaymentID: "cs_100"}
	repo.On("GetByStripeID", ctx, "cs_100").Return(payment, nil)
	repo.On("UpdateStatus", ctx, "pay_1", domain.StatusSucceeded, "pi_200").Return(nil)

	err := uc.ProcessWebhookEvent(ctx, event)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

// --- ListPayments ---

func TestListPayments_Success(t *testing.T) {
	uc, repo, _ := newTestUseCase()
	ctx := context.Background()

	expected := []*domain.Payment{
		{ID: "pay_1", Amount: 1000},
		{ID: "pay_2", Amount: 2000},
	}
	repo.On("List", ctx, 10, 0).Return(expected, nil)

	payments, err := uc.ListPayments(ctx, 10, 0)

	assert.NoError(t, err)
	assert.Len(t, payments, 2)
}

// --- Helper functions ---

func TestMapStripeStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected domain.PaymentStatus
	}{
		{"succeeded", domain.StatusSucceeded},
		{"canceled", domain.StatusCanceled},
		{"requires_payment_method", domain.StatusPending},
		{"requires_confirmation", domain.StatusPending},
		{"requires_action", domain.StatusPending},
		{"processing", domain.StatusPending},
		{"unknown_status", domain.StatusFailed},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, mapStripeStatus(tt.input))
		})
	}
}

func TestJoinMethods(t *testing.T) {
	assert.Equal(t, "", joinMethods(nil))
	assert.Equal(t, "", joinMethods([]domain.PaymentMethodType{}))
	assert.Equal(t, "card", joinMethods([]domain.PaymentMethodType{domain.MethodCard}))
	assert.Equal(t, "card,promptpay", joinMethods([]domain.PaymentMethodType{domain.MethodCard, domain.MethodPromptPay}))
}
