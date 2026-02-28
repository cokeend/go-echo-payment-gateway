package worker

import (
	"encoding/json"
	"fmt"

	"go-payment-gateway/internal/domain"

	"github.com/hibiken/asynq"
)

const (
	TypeWebhookProcess = "webhook:process"
)

// WebhookPayload is the serialized payload for async webhook processing.
type WebhookPayload struct {
	Type            string `json:"type"`
	SessionID       string `json:"session_id,omitempty"`
	PaymentIntentID string `json:"payment_intent_id,omitempty"`
	Status          string `json:"status,omitempty"`
}

// NewWebhookTask creates an asynq task from a verified WebhookEvent.
func NewWebhookTask(event *domain.WebhookEvent) (*asynq.Task, error) {
	payload, err := json.Marshal(WebhookPayload{
		Type:            event.Type,
		SessionID:       event.SessionID,
		PaymentIntentID: event.PaymentIntentID,
		Status:          event.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal webhook payload: %w", err)
	}
	return asynq.NewTask(TypeWebhookProcess, payload), nil
}

// ParseWebhookPayload deserializes the task payload back to a WebhookEvent.
func ParseWebhookPayload(data []byte) (*domain.WebhookEvent, error) {
	var p WebhookPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("unmarshal webhook payload: %w", err)
	}
	return &domain.WebhookEvent{
		Type:            p.Type,
		SessionID:       p.SessionID,
		PaymentIntentID: p.PaymentIntentID,
		Status:          p.Status,
	}, nil
}
