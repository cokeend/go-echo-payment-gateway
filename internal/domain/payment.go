package domain

import (
	"context"
	"time"
)

type PaymentStatus string

const (
	StatusPending   PaymentStatus = "pending"
	StatusSucceeded PaymentStatus = "succeeded"
	StatusFailed    PaymentStatus = "failed"
	StatusCanceled  PaymentStatus = "canceled"
	StatusRefunded  PaymentStatus = "refunded"
)

type PaymentMethodType string

const (
	MethodCard               PaymentMethodType = "card"                 // Visa, Mastercard, etc. (includes Apple Pay / Google Pay)
	MethodPromptPay          PaymentMethodType = "promptpay"            // Thai QR PromptPay (THB only)
	MethodMobileBankingSCB   PaymentMethodType = "mobile_banking_scb"   // SCB Mobile Banking (THB only)
	MethodMobileBankingKBank PaymentMethodType = "mobile_banking_kbank" // KBank Mobile Banking (THB only)
	MethodMobileBankingBBL   PaymentMethodType = "mobile_banking_bbl"   // Bangkok Bank Mobile Banking (THB only)
	MethodMobileBankingBAY   PaymentMethodType = "mobile_banking_bay"   // Krungsri Mobile Banking (THB only)
	MethodMobileBankingKTB   PaymentMethodType = "mobile_banking_ktb"   // Krungthai Mobile Banking (THB only)
	MethodAlipay             PaymentMethodType = "alipay"
	MethodWeChatPay          PaymentMethodType = "wechat_pay"
	MethodGrabPay            PaymentMethodType = "grabpay"
)

type Payment struct {
	ID              string        `json:"id" gorm:"type:varchar(36);primaryKey"`
	StripePaymentID string        `json:"stripe_payment_id" gorm:"type:varchar(255);index"`
	Amount          int64         `json:"amount" gorm:"not null"` // smallest currency unit (e.g. cents)
	Currency        string        `json:"currency" gorm:"type:varchar(3);not null"`
	Status          PaymentStatus `json:"status" gorm:"type:varchar(20);not null;default:'pending';index"`
	PaymentMethod   string        `json:"payment_method,omitempty" gorm:"type:varchar(255)"`
	CustomerEmail   string        `json:"customer_email" gorm:"type:varchar(255);not null"`
	Description     string        `json:"description" gorm:"type:text"`
	ClientSecret    string        `json:"client_secret,omitempty" gorm:"-"`
	CheckoutURL     string        `json:"checkout_url,omitempty" gorm:"type:text"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

type CreatePaymentRequest struct {
	Amount         int64               `json:"amount" validate:"required,gt=0"`
	Currency       string              `json:"currency" validate:"required,len=3"`
	CustomerEmail  string              `json:"customer_email" validate:"required,email"`
	Description    string              `json:"description"`
	PaymentMethods []PaymentMethodType `json:"payment_methods,omitempty"`
}

type CreateCheckoutRequest struct {
	Amount         int64               `json:"amount" validate:"required,gt=0"`
	Currency       string              `json:"currency" validate:"required,len=3"`
	CustomerEmail  string              `json:"customer_email" validate:"required,email"`
	Description    string              `json:"description"`
	SuccessURL     string              `json:"success_url" validate:"required,url"`
	CancelURL      string              `json:"cancel_url" validate:"required,url"`
	PaymentMethods []PaymentMethodType `json:"payment_methods,omitempty"`
}

type RefundRequest struct {
	Amount int64 `json:"amount" validate:"gt=0"`
}

// PaymentIntentResult holds Stripe payment intent response data.
type PaymentIntentResult struct {
	ID           string
	ClientSecret string
	Status       string
}

type RefundResult struct {
	ID     string
	Status string
	Amount int64
}

type CheckoutSessionResult struct {
	SessionID       string
	CheckoutURL     string
	PaymentIntentID string
}

type WebhookEvent struct {
	Type            string
	SessionID       string // Checkout Session ID (cs_xxx)
	PaymentIntentID string // PaymentIntent ID (pi_xxx)
	Status          string
}

// PaymentRepository abstracts persistent storage for payments.
type PaymentRepository interface {
	Create(ctx context.Context, payment *Payment) error
	GetByID(ctx context.Context, id string) (*Payment, error)
	GetByStripeID(ctx context.Context, stripeID string) (*Payment, error)
	UpdateStatus(ctx context.Context, id string, status PaymentStatus, stripePaymentID string) error
	List(ctx context.Context, limit, offset int) ([]*Payment, error)
}

// PaymentGateway abstracts the external payment provider (Stripe).
type PaymentGateway interface {
	CreatePaymentIntent(ctx context.Context, amount int64, currency string, paymentMethods []PaymentMethodType, metadata map[string]string) (*PaymentIntentResult, error)
	CreateCheckoutSession(ctx context.Context, req *CreateCheckoutRequest, metadata map[string]string) (*CheckoutSessionResult, error)
	ConfirmPaymentIntent(ctx context.Context, paymentIntentID string) (*PaymentIntentResult, error)
	CancelPaymentIntent(ctx context.Context, paymentIntentID string) error
	CreateRefund(ctx context.Context, paymentIntentID string, amount int64) (*RefundResult, error)
	ConstructWebhookEvent(payload []byte, signature string) (*WebhookEvent, error)
}

// PaymentUseCase defines the business operations for payments.
type PaymentUseCase interface {
	CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*Payment, error)
	CreateCheckout(ctx context.Context, req *CreateCheckoutRequest) (*Payment, error)
	GetPayment(ctx context.Context, id string) (*Payment, error)
	ConfirmPayment(ctx context.Context, id string) (*Payment, error)
	CancelPayment(ctx context.Context, id string) (*Payment, error)
	RefundPayment(ctx context.Context, id string, amount int64) (*Payment, error)
	HandleWebhook(ctx context.Context, payload []byte, signature string) error
	ListPayments(ctx context.Context, limit, offset int) ([]*Payment, error)

	// VerifyWebhook validates the Stripe signature and returns the parsed event
	// without processing it (for async processing via worker).
	VerifyWebhook(payload []byte, signature string) (*WebhookEvent, error)

	// ProcessWebhookEvent handles a pre-verified webhook event (called by worker).
	ProcessWebhookEvent(ctx context.Context, event *WebhookEvent) error
}
