package worker

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"go-payment-gateway/internal/domain"
	"go-payment-gateway/internal/domain/mocks"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandleWebhookProcess_Success(t *testing.T) {
	uc := new(mocks.MockPaymentUseCase)
	handler := NewTaskHandler(uc)
	ctx := context.Background()

	event := &domain.WebhookEvent{
		Type:            "checkout.session.completed",
		SessionID:       "cs_123",
		PaymentIntentID: "pi_456",
		Status:          "complete",
	}

	payload, _ := json.Marshal(WebhookPayload{
		Type:            event.Type,
		SessionID:       event.SessionID,
		PaymentIntentID: event.PaymentIntentID,
		Status:          event.Status,
	})

	uc.On("ProcessWebhookEvent", mock.Anything, mock.MatchedBy(func(e *domain.WebhookEvent) bool {
		return e.Type == "checkout.session.completed" && e.SessionID == "cs_123"
	})).Return(nil)

	task := asynq.NewTask(TypeWebhookProcess, payload)
	err := handler.HandleWebhookProcess(ctx, task)

	assert.NoError(t, err)
	uc.AssertExpectations(t)
}

func TestHandleWebhookProcess_ProcessError(t *testing.T) {
	uc := new(mocks.MockPaymentUseCase)
	handler := NewTaskHandler(uc)
	ctx := context.Background()

	payload, _ := json.Marshal(WebhookPayload{
		Type:            "payment_intent.succeeded",
		PaymentIntentID: "pi_789",
		Status:          "succeeded",
	})

	uc.On("ProcessWebhookEvent", mock.Anything, mock.Anything).
		Return(errors.New("db connection error"))

	task := asynq.NewTask(TypeWebhookProcess, payload)
	err := handler.HandleWebhookProcess(ctx, task)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "process webhook event")
}

func TestHandleWebhookProcess_InvalidPayload(t *testing.T) {
	uc := new(mocks.MockPaymentUseCase)
	handler := NewTaskHandler(uc)
	ctx := context.Background()

	task := asynq.NewTask(TypeWebhookProcess, []byte("invalid"))
	err := handler.HandleWebhookProcess(ctx, task)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse webhook task")
}
