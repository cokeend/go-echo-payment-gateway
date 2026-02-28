package worker

import (
	"context"
	"fmt"
	"log/slog"

	"go-payment-gateway/internal/domain"

	"github.com/hibiken/asynq"
)

// TaskHandler processes async tasks using the payment use case.
type TaskHandler struct {
	uc domain.PaymentUseCase
}

func NewTaskHandler(uc domain.PaymentUseCase) *TaskHandler {
	return &TaskHandler{uc: uc}
}

// RegisterHandlers wires task types to their handler functions.
func (h *TaskHandler) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TypeWebhookProcess, h.HandleWebhookProcess)
}

func (h *TaskHandler) HandleWebhookProcess(ctx context.Context, task *asynq.Task) error {
	event, err := ParseWebhookPayload(task.Payload())
	if err != nil {
		return fmt.Errorf("parse webhook task: %w", err)
	}

	slog.Info("processing webhook event",
		"type", event.Type,
		"session_id", event.SessionID,
		"payment_intent_id", event.PaymentIntentID,
	)

	if err := h.uc.ProcessWebhookEvent(ctx, event); err != nil {
		return fmt.Errorf("process webhook event: %w", err)
	}

	slog.Info("webhook event processed", "type", event.Type)
	return nil
}
