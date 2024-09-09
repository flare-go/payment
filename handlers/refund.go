package handlers

import (
	"github.com/labstack/echo/v4"
	"goflare.io/payment"
	"net/http"
)

type RefundHandler interface {
	CreateRefund(c echo.Context) error
	GetRefund(c echo.Context) error
	ListRefunds(c echo.Context) error
}

type refundHandler struct {
	Payment payment.Payment
}

func NewRefundHandler(
	Payment payment.Payment,
) RefundHandler {
	return &refundHandler{
		Payment: Payment,
	}
}

// CreateRefund handles POST /refunds
func (rh *refundHandler) CreateRefund(c echo.Context) error {
	var req struct {
		PaymentIntentID string `json:"payment_intent_id"`
		Amount          uint64 `json:"amount"`
		Reason          string `json:"reason"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	if err := rh.Payment.CreateRefund(req.PaymentIntentID, req.Reason, req.Amount); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create refund"})
	}

	return c.NoContent(http.StatusCreated)
}

// GetRefund handles GET /refunds/:id
func (rh *refundHandler) GetRefund(c echo.Context) error {
	id := c.Param("id")

	refund, err := rh.Payment.GetRefund(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Refund not found"})
	}

	return c.JSON(http.StatusOK, refund)
}

// ListRefunds handles GET /refunds
func (rh *refundHandler) ListRefunds(c echo.Context) error {
	paymentIntentID := c.Param("id")

	refunds, err := rh.Payment.ListRefunds(c.Request().Context(), paymentIntentID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list refunds"})
	}

	return c.JSON(http.StatusOK, refunds)
}
