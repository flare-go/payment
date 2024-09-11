package handlers

import (
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"goflare.io/payment"
)

type WebhookHandler interface {
	HandleWebhook(c echo.Context) error
}

type webhookHandler struct {
	Payment payment.Payment
	Logger  *zap.Logger
}

func NewWebhookHandler(payment payment.Payment, logger *zap.Logger) WebhookHandler {
	return &webhookHandler{
		Payment: payment,
		Logger:  logger,
	}
}

// HandleWebhook processes incoming Stripe webhook events
func (wh *webhookHandler) HandleWebhook(c echo.Context) error {
	wh.Logger.Info("Handling webhook")
	payload, err := io.ReadAll(c.Request().Body)
	if err != nil {
		wh.Logger.Error("Failed to read webhook payload", zap.Error(err))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Failed to read request body"})
	}

	signature := c.Request().Header.Get("Stripe-Signature")

	err = wh.Payment.HandleStripeWebhook(c.Request().Context(), payload, signature)
	if err != nil {
		wh.Logger.Error("Failed to handle webhook", zap.Error(err))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Failed to process webhook"})
	}

	return c.NoContent(http.StatusOK)
}
