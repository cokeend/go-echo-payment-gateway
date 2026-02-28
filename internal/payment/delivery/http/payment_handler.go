package http

import (
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"go-payment-gateway/internal/domain"
	apimw "go-payment-gateway/internal/payment/delivery/http/middleware"
	"go-payment-gateway/internal/worker"

	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v5"
)

type PaymentHandler struct {
	uc    domain.PaymentUseCase
	queue *asynq.Client
}

func NewPaymentHandler(uc domain.PaymentUseCase, queue *asynq.Client) *PaymentHandler {
	return &PaymentHandler{uc: uc, queue: queue}
}

// RegisterRoutes wires all payment endpoints into the Echo router.
func (h *PaymentHandler) RegisterRoutes(e *echo.Echo, apiKey string) {
	auth := apimw.APIKeyAuth(apimw.APIKeyConfig{Key: apiKey})

	g := e.Group("/api/v1/payments", auth)
	g.POST("", h.CreatePayment)
	g.GET("", h.ListPayments)
	g.GET("/:id", h.GetPayment)
	g.POST("/:id/confirm", h.ConfirmPayment)
	g.POST("/:id/cancel", h.CancelPayment)
	g.POST("/:id/refund", h.RefundPayment)

	e.POST("/api/v1/checkout", h.CreateCheckout, auth)

	// Webhook uses Stripe signature verification, no API key needed
	e.POST("/api/v1/webhook/stripe", h.HandleStripeWebhook)
}

type apiResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func (h *PaymentHandler) CreatePayment(c *echo.Context) error {
	var req domain.CreatePaymentRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, apiResponse{Error: "invalid request body"})
	}

	if req.Amount <= 0 {
		return c.JSON(http.StatusBadRequest, apiResponse{Error: "amount must be greater than 0"})
	}
	if req.Currency == "" {
		return c.JSON(http.StatusBadRequest, apiResponse{Error: "currency is required"})
	}
	if req.CustomerEmail == "" {
		return c.JSON(http.StatusBadRequest, apiResponse{Error: "customer_email is required"})
	}

	payment, err := h.uc.CreatePayment(c.Request().Context(), &req)
	if err != nil {
		slog.Error("create payment failed", "error", err)
		return c.JSON(http.StatusInternalServerError, apiResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusCreated, apiResponse{Success: true, Data: payment})
}

func (h *PaymentHandler) CreateCheckout(c *echo.Context) error {
	var req domain.CreateCheckoutRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, apiResponse{Error: "invalid request body"})
	}

	if req.Amount <= 0 {
		return c.JSON(http.StatusBadRequest, apiResponse{Error: "amount must be greater than 0"})
	}
	if req.Currency == "" {
		return c.JSON(http.StatusBadRequest, apiResponse{Error: "currency is required"})
	}
	if req.CustomerEmail == "" {
		return c.JSON(http.StatusBadRequest, apiResponse{Error: "customer_email is required"})
	}
	if req.SuccessURL == "" || req.CancelURL == "" {
		return c.JSON(http.StatusBadRequest, apiResponse{Error: "success_url and cancel_url are required"})
	}

	payment, err := h.uc.CreateCheckout(c.Request().Context(), &req)
	if err != nil {
		slog.Error("create checkout failed", "error", err)
		return c.JSON(http.StatusInternalServerError, apiResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusCreated, apiResponse{Success: true, Data: payment})
}

func (h *PaymentHandler) GetPayment(c *echo.Context) error {
	id := c.Param("id")

	payment, err := h.uc.GetPayment(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, apiResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, apiResponse{Success: true, Data: payment})
}

func (h *PaymentHandler) ConfirmPayment(c *echo.Context) error {
	id := c.Param("id")

	payment, err := h.uc.ConfirmPayment(c.Request().Context(), id)
	if err != nil {
		slog.Error("confirm payment failed", "error", err)
		return c.JSON(http.StatusBadRequest, apiResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, apiResponse{Success: true, Data: payment})
}

func (h *PaymentHandler) CancelPayment(c *echo.Context) error {
	id := c.Param("id")

	payment, err := h.uc.CancelPayment(c.Request().Context(), id)
	if err != nil {
		slog.Error("cancel payment failed", "error", err)
		return c.JSON(http.StatusBadRequest, apiResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, apiResponse{Success: true, Data: payment})
}

func (h *PaymentHandler) RefundPayment(c *echo.Context) error {
	id := c.Param("id")

	var req domain.RefundRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, apiResponse{Error: "invalid request body"})
	}

	payment, err := h.uc.RefundPayment(c.Request().Context(), id, req.Amount)
	if err != nil {
		slog.Error("refund payment failed", "error", err)
		return c.JSON(http.StatusBadRequest, apiResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, apiResponse{Success: true, Data: payment})
}

func (h *PaymentHandler) ListPayments(c *echo.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	if limit == 0 {
		limit = 20
	}

	payments, err := h.uc.ListPayments(c.Request().Context(), limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, apiResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, apiResponse{Success: true, Data: payments})
}

func (h *PaymentHandler) HandleStripeWebhook(c *echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, apiResponse{Error: "failed to read request body"})
	}

	signature := c.Request().Header.Get("Stripe-Signature")
	if signature == "" {
		return c.JSON(http.StatusBadRequest, apiResponse{Error: "missing Stripe-Signature header"})
	}

	// Verify signature immediately, reject invalid payloads
	event, err := h.uc.VerifyWebhook(body, signature)
	if err != nil {
		slog.Error("webhook verification failed", "error", err)
		return c.JSON(http.StatusBadRequest, apiResponse{Error: err.Error()})
	}

	// Enqueue for async processing by worker
	task, err := worker.NewWebhookTask(event)
	if err != nil {
		slog.Error("failed to create webhook task", "error", err)
		return c.JSON(http.StatusInternalServerError, apiResponse{Error: "failed to enqueue webhook"})
	}

	info, err := h.queue.Enqueue(task,
		asynq.MaxRetry(5),
		asynq.Timeout(30*time.Second),
		asynq.Queue("critical"),
	)
	if err != nil {
		slog.Error("failed to enqueue webhook task", "error", err)
		// Fallback: process synchronously if queue is unavailable
		if procErr := h.uc.ProcessWebhookEvent(c.Request().Context(), event); procErr != nil {
			slog.Error("fallback webhook processing failed", "error", procErr)
			return c.JSON(http.StatusInternalServerError, apiResponse{Error: procErr.Error()})
		}
		return c.JSON(http.StatusOK, apiResponse{Success: true})
	}

	slog.Info("webhook task enqueued", "task_id", info.ID, "queue", info.Queue, "type", event.Type)
	return c.JSON(http.StatusOK, apiResponse{Success: true})
}
