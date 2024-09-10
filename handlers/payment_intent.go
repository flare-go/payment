package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/stripe/stripe-go/v79"
	"net/http"

	"goflare.io/payment"
)

type PaymentIntentHandler interface {
	CreatePaymentIntent(c echo.Context) error
	GetPaymentIntent(c echo.Context) error
	ConfirmPaymentIntent(c echo.Context) error
	CancelPaymentIntent(c echo.Context) error
	ListPaymentIntents(c echo.Context) error
	ListPaymentIntentsByCustomer(c echo.Context) error
}

type paymentIntentHandler struct {
	Payment payment.Payment
}

func NewPaymentIntentHandler(Payment payment.Payment) PaymentIntentHandler {
	return &paymentIntentHandler{
		Payment: Payment,
	}
}

// CreatePaymentIntent handles POST /payment_intents
func (ph *paymentIntentHandler) CreatePaymentIntent(c echo.Context) error {
	var req struct {
		CustomerID      string          `json:"customer_id"`
		Amount          uint64          `json:"amount"`
		Currency        stripe.Currency `json:"currency"`
		PaymentMethodID string          `json:"payment_method_id,omitempty"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	if err := ph.Payment.CreatePaymentIntent(req.CustomerID, req.PaymentMethodID, req.Amount, req.Currency); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create payment intent"})
	}

	return c.NoContent(http.StatusCreated)
}

// GetPaymentIntent handles GET /payment_intents/:id
func (ph *paymentIntentHandler) GetPaymentIntent(c echo.Context) error {
	id := c.Param("id")

	paymentIntent, err := ph.Payment.GetPaymentIntent(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Payment intent not found"})
	}

	return c.JSON(http.StatusOK, paymentIntent)
}

// ConfirmPaymentIntent handles POST /payment_intents/:id/confirm
func (ph *paymentIntentHandler) ConfirmPaymentIntent(c echo.Context) error {

	var req struct {
		PaymentMethodID string `json:"payment_method_id"`
		PaymentIntentID string `json:"payment_intent_id"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	if err := ph.Payment.ConfirmPaymentIntent(req.PaymentIntentID, req.PaymentMethodID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to confirm payment intent"})
	}

	return c.NoContent(http.StatusOK)
}

// CancelPaymentIntent handles POST /payment_intents/:id/cancel
func (ph *paymentIntentHandler) CancelPaymentIntent(c echo.Context) error {
	id := c.Param("id")

	if err := ph.Payment.CancelPaymentIntent(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to cancel payment intent"})
	}

	return c.NoContent(http.StatusOK)
}

// ListPaymentIntents handles GET /payment_intents
func (ph *paymentIntentHandler) ListPaymentIntents(c echo.Context) error {
	var req struct {
		Limit  uint64 `json:"limit"`
		Offset uint64 `json:"offset"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	if req.Limit == 0 {
		req.Limit = 10 // 默認限制
	}

	paymentIntents, err := ph.Payment.ListPaymentIntent(c.Request().Context(), req.Limit, req.Offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list payment intents"})
	}

	return c.JSON(http.StatusOK, paymentIntents)
}

func (ph *paymentIntentHandler) ListPaymentIntentsByCustomer(c echo.Context) error {
	var req struct {
		CustomerID string `json:"customer_id"`
		Limit      uint64 `json:"limit"`
		Offset     uint64 `json:"offset"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	if req.Limit == 0 {
		req.Limit = 10 // 默認限制
	}

	paymentIntents, err := ph.Payment.ListPaymentIntentByCustomerID(c.Request().Context(), req.CustomerID, req.Limit, req.Offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list payment intents"})
	}

	return c.JSON(http.StatusOK, paymentIntents)
}
