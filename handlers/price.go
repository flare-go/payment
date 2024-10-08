package handlers

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/stripe/stripe-go/v79"
	"go.uber.org/zap"

	"goflare.io/payment"
	"goflare.io/payment/models"
)

type PriceHandler interface {
	CreatePrice(c echo.Context) error
	DeletePrice(c echo.Context) error
}

type priceHandler struct {
	Payment payment.Payment
	Logger  *zap.Logger
}

func NewPriceHandler(payment payment.Payment, logger *zap.Logger) PriceHandler {
	return &priceHandler{
		Payment: payment,
		Logger:  logger,
	}
}

func (ph *priceHandler) CreatePrice(c echo.Context) error {

	var req models.Price
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	if err := validateCreatePriceRequest(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if err := ph.Payment.CreatePrice(req); err != nil {
		ph.Logger.Error("Failed to create price", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create price"})
	}

	return c.NoContent(http.StatusCreated)
}

func (ph *priceHandler) DeletePrice(c echo.Context) error {

	id := c.Param("id")

	if err := ph.Payment.DeletePrice(id); err != nil {
		ph.Logger.Error("Failed to delete price", zap.Error(err), zap.String("id", id))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete price"})
	}

	return c.NoContent(http.StatusNoContent)
}

func validateCreatePriceRequest(req models.Price) error {
	if len(req.ProductID) == 0 {
		return errors.New("product_id is required")
	}
	if req.UnitAmount <= 0 {
		return errors.New("unit_amount must be greater than 0")
	}
	if req.Type == stripe.PriceTypeRecurring {
		if req.RecurringInterval == "" {
			return errors.New("recurring_interval is required for recurring prices")
		}
		if req.RecurringIntervalCount <= 0 {
			return errors.New("recurring_interval_count must be greater than 0 for recurring prices")
		}
	}
	return nil
}
