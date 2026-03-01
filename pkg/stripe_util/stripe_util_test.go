package stripe_util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSupportedCurrency(t *testing.T) {
	supported := []string{"usd", "eur", "gbp", "jpy", "thb", "sgd", "aud", "cad", "myr", "cny", "hkd"}
	for _, c := range supported {
		assert.True(t, SupportedCurrency(c), "expected %s to be supported", c)
	}

	unsupported := []string{"xyz", "abc", "krw", "THB", "USD", ""}
	for _, c := range unsupported {
		assert.False(t, SupportedCurrency(c), "expected %s to be unsupported", c)
	}
}

func TestValidPaymentMethod(t *testing.T) {
	valid := []string{
		"card", "promptpay", "mobile_banking_scb", "mobile_banking_kbank",
		"mobile_banking_bbl", "mobile_banking_bay", "mobile_banking_ktb",
		"alipay", "wechat_pay", "grabpay",
	}
	for _, m := range valid {
		assert.True(t, ValidPaymentMethod(m), "expected %s to be valid", m)
	}

	invalid := []string{"bitcoin", "paypal", "bank_transfer", ""}
	for _, m := range invalid {
		assert.False(t, ValidPaymentMethod(m), "expected %s to be invalid", m)
	}
}

func TestValidateMethodCurrency(t *testing.T) {
	tests := []struct {
		method   string
		currency string
		wantErr  bool
	}{
		{"card", "usd", false},
		{"card", "thb", false},
		{"promptpay", "thb", false},
		{"promptpay", "usd", true},
		{"mobile_banking_scb", "thb", false},
		{"mobile_banking_scb", "usd", true},
		{"mobile_banking_kbank", "thb", false},
		{"mobile_banking_kbank", "eur", true},
		{"mobile_banking_bbl", "thb", false},
		{"mobile_banking_bay", "thb", false},
		{"mobile_banking_ktb", "thb", false},
		{"grabpay", "sgd", false},
		{"grabpay", "myr", false},
		{"grabpay", "thb", true},
		{"grabpay", "usd", true},
		{"alipay", "usd", false},
		{"alipay", "thb", false},
		{"wechat_pay", "usd", false},
	}

	for _, tt := range tests {
		name := tt.method + "_" + tt.currency
		t.Run(name, func(t *testing.T) {
			msg := ValidateMethodCurrency(tt.method, tt.currency)
			if tt.wantErr {
				assert.NotEmpty(t, msg, "expected error for %s with %s", tt.method, tt.currency)
			} else {
				assert.Empty(t, msg, "expected no error for %s with %s", tt.method, tt.currency)
			}
		})
	}
}

func TestCentsToDollars(t *testing.T) {
	assert.Equal(t, "10.00", CentsToDollars(1000))
	assert.Equal(t, "0.50", CentsToDollars(50))
	assert.Equal(t, "0.01", CentsToDollars(1))
	assert.Equal(t, "0.00", CentsToDollars(0))
	assert.Equal(t, "123.45", CentsToDollars(12345))
}
