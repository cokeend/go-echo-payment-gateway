package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"go-payment-gateway/internal/domain"
	"go-payment-gateway/pkg/stripe_util"
)

type paymentUseCase struct {
	repo    domain.PaymentRepository
	gateway domain.PaymentGateway
}

func NewPaymentUseCase(repo domain.PaymentRepository, gw domain.PaymentGateway) domain.PaymentUseCase {
	return &paymentUseCase{repo: repo, gateway: gw}
}

func (uc *paymentUseCase) CreatePayment(ctx context.Context, req *domain.CreatePaymentRequest) (*domain.Payment, error) {
	currency := strings.ToLower(req.Currency)
	if !stripe_util.SupportedCurrency(currency) {
		return nil, fmt.Errorf("unsupported currency: %s", req.Currency)
	}

	if err := validatePaymentMethods(req.PaymentMethods, currency); err != nil {
		return nil, err
	}

	metadata := map[string]string{
		"customer_email": req.CustomerEmail,
	}
	if req.Description != "" {
		metadata["description"] = req.Description
	}

	result, err := uc.gateway.CreatePaymentIntent(ctx, req.Amount, currency, req.PaymentMethods, metadata)
	if err != nil {
		return nil, fmt.Errorf("create payment intent: %w", err)
	}

	payment := &domain.Payment{
		StripePaymentID: result.ID,
		Amount:          req.Amount,
		Currency:        currency,
		Status:          domain.StatusPending,
		PaymentMethod:   joinMethods(req.PaymentMethods),
		CustomerEmail:   req.CustomerEmail,
		Description:     req.Description,
		ClientSecret:    result.ClientSecret,
	}

	if err := uc.repo.Create(ctx, payment); err != nil {
		return nil, fmt.Errorf("save payment: %w", err)
	}

	return payment, nil
}

func (uc *paymentUseCase) CreateCheckout(ctx context.Context, req *domain.CreateCheckoutRequest) (*domain.Payment, error) {
	currency := strings.ToLower(req.Currency)
	if !stripe_util.SupportedCurrency(currency) {
		return nil, fmt.Errorf("unsupported currency: %s", req.Currency)
	}
	req.Currency = currency

	if err := validatePaymentMethods(req.PaymentMethods, currency); err != nil {
		return nil, err
	}

	if req.Description == "" {
		req.Description = "Payment"
	}

	metadata := map[string]string{
		"customer_email": req.CustomerEmail,
	}

	result, err := uc.gateway.CreateCheckoutSession(ctx, req, metadata)
	if err != nil {
		return nil, fmt.Errorf("create checkout session: %w", err)
	}

	stripeID := result.PaymentIntentID
	if stripeID == "" {
		stripeID = result.SessionID // use session ID until PaymentIntent is created
	}

	payment := &domain.Payment{
		StripePaymentID: stripeID,
		Amount:          req.Amount,
		Currency:        currency,
		Status:          domain.StatusPending,
		PaymentMethod:   joinMethods(req.PaymentMethods),
		CustomerEmail:   req.CustomerEmail,
		Description:     req.Description,
		CheckoutURL:     result.CheckoutURL,
	}

	if err := uc.repo.Create(ctx, payment); err != nil {
		return nil, fmt.Errorf("save payment: %w", err)
	}

	return payment, nil
}

func (uc *paymentUseCase) GetPayment(ctx context.Context, id string) (*domain.Payment, error) {
	return uc.repo.GetByID(ctx, id)
}

func (uc *paymentUseCase) ConfirmPayment(ctx context.Context, id string) (*domain.Payment, error) {
	payment, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if payment.Status != domain.StatusPending {
		return nil, fmt.Errorf("payment %s is not in pending state (current: %s)", id, payment.Status)
	}

	result, err := uc.gateway.ConfirmPaymentIntent(ctx, payment.StripePaymentID)
	if err != nil {
		return nil, fmt.Errorf("confirm payment intent: %w", err)
	}

	newStatus := mapStripeStatus(result.Status)
	if err := uc.repo.UpdateStatus(ctx, id, newStatus, payment.StripePaymentID); err != nil {
		return nil, fmt.Errorf("update payment status: %w", err)
	}

	payment.Status = newStatus
	return payment, nil
}

func (uc *paymentUseCase) CancelPayment(ctx context.Context, id string) (*domain.Payment, error) {
	payment, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if payment.Status != domain.StatusPending {
		return nil, fmt.Errorf("only pending payments can be canceled (current: %s)", payment.Status)
	}

	if err := uc.gateway.CancelPaymentIntent(ctx, payment.StripePaymentID); err != nil {
		return nil, fmt.Errorf("cancel payment intent: %w", err)
	}

	if err := uc.repo.UpdateStatus(ctx, id, domain.StatusCanceled, payment.StripePaymentID); err != nil {
		return nil, fmt.Errorf("update payment status: %w", err)
	}

	payment.Status = domain.StatusCanceled
	return payment, nil
}

func (uc *paymentUseCase) RefundPayment(ctx context.Context, id string, amount int64) (*domain.Payment, error) {
	payment, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if payment.Status != domain.StatusSucceeded {
		return nil, fmt.Errorf("only succeeded payments can be refunded (current: %s)", payment.Status)
	}

	if amount > payment.Amount {
		return nil, fmt.Errorf("refund amount (%d) exceeds payment amount (%d)", amount, payment.Amount)
	}

	_, err = uc.gateway.CreateRefund(ctx, payment.StripePaymentID, amount)
	if err != nil {
		return nil, fmt.Errorf("create refund: %w", err)
	}

	if err := uc.repo.UpdateStatus(ctx, id, domain.StatusRefunded, payment.StripePaymentID); err != nil {
		return nil, fmt.Errorf("update payment status: %w", err)
	}

	payment.Status = domain.StatusRefunded
	return payment, nil
}

func (uc *paymentUseCase) HandleWebhook(ctx context.Context, payload []byte, signature string) error {
	event, err := uc.gateway.ConstructWebhookEvent(payload, signature)
	if err != nil {
		return fmt.Errorf("construct webhook event: %w", err)
	}

	switch event.Type {
	case "checkout.session.completed", "checkout.session.async_payment_succeeded":
		return uc.handleCheckoutSessionEvent(ctx, event)
	case "payment_intent.succeeded", "payment_intent.payment_failed", "payment_intent.canceled":
		return uc.handlePaymentIntentEvent(ctx, event)
	default:
		slog.Info("webhook event ignored", "type", event.Type)
		return nil
	}
}

func (uc *paymentUseCase) handleCheckoutSessionEvent(ctx context.Context, event *domain.WebhookEvent) error {
	if event.SessionID == "" {
		return nil
	}

	// find payment by session ID (cs_xxx) stored during checkout creation
	payment, err := uc.repo.GetByStripeID(ctx, event.SessionID)
	if err != nil {
		return fmt.Errorf("find payment for checkout session: %w", err)
	}

	// update with the actual PaymentIntent ID + mark succeeded
	piID := event.PaymentIntentID
	if piID == "" {
		piID = event.SessionID
	}

	slog.Info("checkout session completed",
		"payment_id", payment.ID,
		"session_id", event.SessionID,
		"payment_intent_id", piID,
	)

	return uc.repo.UpdateStatus(ctx, payment.ID, domain.StatusSucceeded, piID)
}

func (uc *paymentUseCase) handlePaymentIntentEvent(ctx context.Context, event *domain.WebhookEvent) error {
	if event.PaymentIntentID == "" {
		return nil
	}

	payment, err := uc.repo.GetByStripeID(ctx, event.PaymentIntentID)
	if err != nil {
		slog.Warn("payment not found for payment intent event, skipping",
			"payment_intent_id", event.PaymentIntentID,
			"type", event.Type,
		)
		return nil
	}

	newStatus := mapStripeStatus(event.Status)
	if newStatus == payment.Status {
		return nil
	}

	slog.Info("payment intent status updated",
		"payment_id", payment.ID,
		"old_status", payment.Status,
		"new_status", newStatus,
		"event_type", event.Type,
	)

	return uc.repo.UpdateStatus(ctx, payment.ID, newStatus, event.PaymentIntentID)
}

func (uc *paymentUseCase) ListPayments(ctx context.Context, limit, offset int) ([]*domain.Payment, error) {
	return uc.repo.List(ctx, limit, offset)
}

func validatePaymentMethods(methods []domain.PaymentMethodType, currency string) error {
	for _, m := range methods {
		method := string(m)
		if !stripe_util.ValidPaymentMethod(method) {
			return fmt.Errorf("unsupported payment method: %s", method)
		}
		if msg := stripe_util.ValidateMethodCurrency(method, currency); msg != "" {
			return fmt.Errorf(msg)
		}
	}
	return nil
}

func joinMethods(methods []domain.PaymentMethodType) string {
	if len(methods) == 0 {
		return ""
	}
	parts := make([]string, len(methods))
	for i, m := range methods {
		parts[i] = string(m)
	}
	return strings.Join(parts, ",")
}

func mapStripeStatus(stripeStatus string) domain.PaymentStatus {
	switch stripeStatus {
	case "succeeded":
		return domain.StatusSucceeded
	case "canceled":
		return domain.StatusCanceled
	case "requires_payment_method", "requires_confirmation", "requires_action", "processing":
		return domain.StatusPending
	default:
		return domain.StatusFailed
	}
}
