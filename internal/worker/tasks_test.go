package worker

import (
	"testing"

	"go-payment-gateway/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWebhookTask_And_Parse(t *testing.T) {
	event := &domain.WebhookEvent{
		Type:            "checkout.session.completed",
		SessionID:       "cs_123",
		PaymentIntentID: "pi_456",
		Status:          "complete",
	}

	task, err := NewWebhookTask(event)
	require.NoError(t, err)
	assert.Equal(t, TypeWebhookProcess, task.Type())

	parsed, err := ParseWebhookPayload(task.Payload())
	require.NoError(t, err)
	assert.Equal(t, event.Type, parsed.Type)
	assert.Equal(t, event.SessionID, parsed.SessionID)
	assert.Equal(t, event.PaymentIntentID, parsed.PaymentIntentID)
	assert.Equal(t, event.Status, parsed.Status)
}

func TestNewWebhookTask_PaymentIntentEvent(t *testing.T) {
	event := &domain.WebhookEvent{
		Type:            "payment_intent.succeeded",
		PaymentIntentID: "pi_789",
		Status:          "succeeded",
	}

	task, err := NewWebhookTask(event)
	require.NoError(t, err)

	parsed, err := ParseWebhookPayload(task.Payload())
	require.NoError(t, err)
	assert.Equal(t, "payment_intent.succeeded", parsed.Type)
	assert.Equal(t, "pi_789", parsed.PaymentIntentID)
	assert.Empty(t, parsed.SessionID)
}

func TestNewWebhookTask_EmptyEvent(t *testing.T) {
	event := &domain.WebhookEvent{Type: "unknown.event"}

	task, err := NewWebhookTask(event)
	require.NoError(t, err)

	parsed, err := ParseWebhookPayload(task.Payload())
	require.NoError(t, err)
	assert.Equal(t, "unknown.event", parsed.Type)
}

func TestParseWebhookPayload_InvalidJSON(t *testing.T) {
	_, err := ParseWebhookPayload([]byte("not json"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal webhook payload")
}
