package gateway

import (
	"context"
	"fmt"
	"strings"

	"go-payment-gateway/internal/domain"
	"go-payment-gateway/pkg/stripe_util"

	"github.com/stripe/stripe-go/v84"
)

type StripeGateway struct {
	client        *stripe.Client
	webhookSecret string
}

func NewStripeGateway(apiKey, webhookSecret string) *StripeGateway {
	// Retry up to 5 times on 429 (rate limit), 500, 503 with exponential backoff
	backend := stripe.GetBackendWithConfig(stripe.APIBackend, &stripe.BackendConfig{
		MaxNetworkRetries: stripe.Int64(5),
	})
	client := stripe.NewClient(apiKey, stripe.WithBackends(&stripe.Backends{
		API:     backend,
		Connect: backend,
		Uploads: backend,
	}))
	return &StripeGateway{
		client:        client,
		webhookSecret: webhookSecret,
	}
}

func (g *StripeGateway) CreatePaymentIntent(ctx context.Context, amount int64, currency string, paymentMethods []domain.PaymentMethodType, metadata map[string]string) (*domain.PaymentIntentResult, error) {
	params := &stripe.PaymentIntentCreateParams{
		Amount:   stripe.Int64(amount),
		Currency: stripe.String(currency),
	}

	if len(paymentMethods) > 0 {
		pmTypes := make([]*string, len(paymentMethods))
		for i, m := range paymentMethods {
			pmTypes[i] = stripe.String(string(m))
		}
		params.PaymentMethodTypes = pmTypes
	} else {
		params.AutomaticPaymentMethods = &stripe.PaymentIntentCreateAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		}
	}

	if len(metadata) > 0 {
		params.Metadata = make(map[string]string)
		for k, v := range metadata {
			params.Metadata[k] = v
		}
	}

	pi, err := g.client.V1PaymentIntents.Create(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("stripe create payment intent: %w", err)
	}

	return &domain.PaymentIntentResult{
		ID:           pi.ID,
		ClientSecret: pi.ClientSecret,
		Status:       string(pi.Status),
	}, nil
}

func (g *StripeGateway) CreateCheckoutSession(ctx context.Context, req *domain.CreateCheckoutRequest, metadata map[string]string) (*domain.CheckoutSessionResult, error) {
	params := &stripe.CheckoutSessionCreateParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionCreateLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionCreateLineItemPriceDataParams{
					Currency:   stripe.String(req.Currency),
					UnitAmount: stripe.Int64(req.Amount),
					ProductData: &stripe.CheckoutSessionCreateLineItemPriceDataProductDataParams{
						Name: stripe.String(req.Description),
					},
				},
				Quantity: stripe.Int64(1),
			},
		},
		CustomerEmail: stripe.String(req.CustomerEmail),
		SuccessURL:    stripe.String(req.SuccessURL),
		CancelURL:     stripe.String(req.CancelURL),
	}

	if len(req.PaymentMethods) > 0 {
		pmTypes := make([]*string, len(req.PaymentMethods))
		for i, m := range req.PaymentMethods {
			pmTypes[i] = stripe.String(string(m))
		}
		params.PaymentMethodTypes = pmTypes
	}

	if len(metadata) > 0 {
		params.Metadata = make(map[string]string)
		for k, v := range metadata {
			params.Metadata[k] = v
		}
	}

	session, err := g.client.V1CheckoutSessions.Create(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("stripe create checkout session: %w", err)
	}

	piID := ""
	if session.PaymentIntent != nil {
		piID = session.PaymentIntent.ID
	}

	return &domain.CheckoutSessionResult{
		SessionID:       session.ID,
		CheckoutURL:     session.URL,
		PaymentIntentID: piID, // empty until customer completes payment
	}, nil
}

func (g *StripeGateway) ConfirmPaymentIntent(ctx context.Context, paymentIntentID string) (*domain.PaymentIntentResult, error) {
	pi, err := g.client.V1PaymentIntents.Confirm(ctx, paymentIntentID, &stripe.PaymentIntentConfirmParams{})
	if err != nil {
		return nil, fmt.Errorf("stripe confirm payment intent: %w", err)
	}

	return &domain.PaymentIntentResult{
		ID:           pi.ID,
		ClientSecret: pi.ClientSecret,
		Status:       string(pi.Status),
	}, nil
}

func (g *StripeGateway) CancelPaymentIntent(ctx context.Context, paymentIntentID string) error {
	_, err := g.client.V1PaymentIntents.Cancel(ctx, paymentIntentID, &stripe.PaymentIntentCancelParams{})
	if err != nil {
		return fmt.Errorf("stripe cancel payment intent: %w", err)
	}
	return nil
}

func (g *StripeGateway) CreateRefund(ctx context.Context, paymentIntentID string, amount int64) (*domain.RefundResult, error) {
	params := &stripe.RefundCreateParams{
		PaymentIntent: stripe.String(paymentIntentID),
	}
	if amount > 0 {
		params.Amount = stripe.Int64(amount)
	}

	refund, err := g.client.V1Refunds.Create(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("stripe create refund: %w", err)
	}

	return &domain.RefundResult{
		ID:     refund.ID,
		Status: string(refund.Status),
		Amount: refund.Amount,
	}, nil
}

func (g *StripeGateway) ConstructWebhookEvent(payload []byte, signature string) (*domain.WebhookEvent, error) {
	event, err := stripe_util.VerifyWebhookSignature(payload, signature, g.webhookSecret)
	if err != nil {
		return nil, err
	}

	we := &domain.WebhookEvent{
		Type: string(event.Type),
	}

	switch {
	case strings.HasPrefix(string(event.Type), "checkout.session."):
		cs, err := stripe_util.ExtractCheckoutSessionFromEvent(event)
		if err == nil && cs != nil {
			we.SessionID = cs.ID
			if cs.PaymentIntent != nil {
				we.PaymentIntentID = cs.PaymentIntent.ID
			}
			we.Status = string(cs.Status)
		}
	case strings.HasPrefix(string(event.Type), "payment_intent."):
		pi, err := stripe_util.ExtractPaymentIntentFromEvent(event)
		if err == nil && pi != nil {
			we.PaymentIntentID = pi.ID
			we.Status = string(pi.Status)
		}
	}

	return we, nil
}
