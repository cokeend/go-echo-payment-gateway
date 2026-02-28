package stripe_util

import (
	"encoding/json"
	"fmt"

	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/webhook"
)

// VerifyWebhookSignature validates the Stripe webhook payload and returns the parsed event.
func VerifyWebhookSignature(payload []byte, sigHeader, webhookSecret string) (*stripe.Event, error) {
	event, err := webhook.ConstructEvent(payload, sigHeader, webhookSecret)
	if err != nil {
		return nil, fmt.Errorf("webhook signature verification failed: %w", err)
	}
	return &event, nil
}

// ExtractPaymentIntentFromEvent extracts payment intent data from a webhook event.
func ExtractPaymentIntentFromEvent(event *stripe.Event) (*stripe.PaymentIntent, error) {
	var pi stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payment intent: %w", err)
	}
	return &pi, nil
}

// ExtractCheckoutSessionFromEvent extracts checkout session data from a webhook event.
func ExtractCheckoutSessionFromEvent(event *stripe.Event) (*stripe.CheckoutSession, error) {
	var cs stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &cs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal checkout session: %w", err)
	}
	return &cs, nil
}

// CentsToDollars converts amount in cents to a human-readable dollar string.
func CentsToDollars(cents int64) string {
	return fmt.Sprintf("%.2f", float64(cents)/100)
}

// SupportedCurrency returns true if the currency is supported.
func SupportedCurrency(currency string) bool {
	supported := map[string]bool{
		"usd": true, "eur": true, "gbp": true, "jpy": true,
		"thb": true, "sgd": true, "aud": true, "cad": true,
		"myr": true, "cny": true, "hkd": true,
	}
	return supported[currency]
}

var allPaymentMethods = map[string]bool{
	"card":                 true,
	"promptpay":            true,
	"mobile_banking_scb":   true,
	"mobile_banking_kbank": true,
	"mobile_banking_bbl":   true,
	"mobile_banking_bay":   true,
	"mobile_banking_ktb":   true,
	"alipay":               true,
	"wechat_pay":           true,
	"grabpay":              true,
}

// THB-only payment methods
var thbOnlyMethods = map[string]bool{
	"promptpay":            true,
	"mobile_banking_scb":   true,
	"mobile_banking_kbank": true,
	"mobile_banking_bbl":   true,
	"mobile_banking_bay":   true,
	"mobile_banking_ktb":   true,
}

// ValidPaymentMethod returns true if the method is recognized.
func ValidPaymentMethod(method string) bool {
	return allPaymentMethods[method]
}

// ValidateMethodCurrency checks that a payment method is compatible with the given currency.
// Returns an error message if incompatible, empty string if ok.
func ValidateMethodCurrency(method, currency string) string {
	if thbOnlyMethods[method] && currency != "thb" {
		return fmt.Sprintf("%s requires currency THB", method)
	}
	if method == "grabpay" && currency != "sgd" && currency != "myr" {
		return "grabpay requires currency SGD or MYR"
	}
	return ""
}
