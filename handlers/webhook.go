package handlers

import (
	"github.com/labstack/echo/v4"
	"goflare.io/payment"
	"io"
	"net/http"
)

type WebhookHandler interface {
	HandleStripeWebhook(c echo.Context) error
}

type webhookHandler struct {
	Payment payment.Payment
}

func NewWebhookHandler(
	Payment payment.Payment,
) WebhookHandler {
	return &webhookHandler{
		Payment: Payment,
	}
}

func (wh *webhookHandler) HandleStripeWebhook(c echo.Context) error {
	payload, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Failed to read request body"})
	}

	signature := c.Request().Header.Get("Stripe-Signature")

	err = wh.Payment.HandleStripeWebhook(c.Request().Context(), payload, signature)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Failed to handle webhook"})
	}

	return c.NoContent(http.StatusOK)
}
