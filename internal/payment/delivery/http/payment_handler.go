package http

import (
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"go-payment-gateway/internal/domain"

	"github.com/labstack/echo/v5"
)

type PaymentHandler struct {
	uc domain.PaymentUseCase
}

func NewPaymentHandler(uc domain.PaymentUseCase) *PaymentHandler {
	return &PaymentHandler{uc: uc}
}

// RegisterRoutes wires all payment endpoints into the Echo router.
func (h *PaymentHandler) RegisterRoutes(e *echo.Echo) {
	g := e.Group("/api/v1/payments")

	g.POST("", h.CreatePayment)
	g.GET("", h.ListPayments)
	g.GET("/:id", h.GetPayment)
	g.POST("/:id/confirm", h.ConfirmPayment)
	g.POST("/:id/cancel", h.CancelPayment)
	g.POST("/:id/refund", h.RefundPayment)

	e.POST("/api/v1/checkout", h.CreateCheckout)
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

	if err := h.uc.HandleWebhook(c.Request().Context(), body, signature); err != nil {
		slog.Error("webhook handling failed", "error", err)
		return c.JSON(http.StatusBadRequest, apiResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, apiResponse{Success: true})
}
