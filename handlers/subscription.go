package handlers

import (
	"github.com/labstack/echo/v4"
	"goflare.io/payment"
	"goflare.io/payment/models"
	"net/http"
)

type SubscriptionHandler interface {
	CreateSubscription(c echo.Context) error
	GetSubscription(c echo.Context) error
	UpdateSubscription(c echo.Context) error
	CancelSubscription(c echo.Context) error
	ListSubscriptions(c echo.Context) error
}

type subscriptionHandler struct {
	Payment payment.Payment
}

func NewSubscriptionHandler(
	Payment payment.Payment,
) SubscriptionHandler {
	return &subscriptionHandler{
		Payment: Payment,
	}
}

// CreateSubscription handles POST /subscriptions
func (sh *subscriptionHandler) CreateSubscription(c echo.Context) error {
	var req struct {
		CustomerID string `json:"customer_id"`
		PriceID    string `json:"price_id"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	if err := sh.Payment.CreateSubscription(req.CustomerID, req.PriceID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create subscription"})
	}

	return c.NoContent(http.StatusCreated)
}

// GetSubscription handles GET /subscriptions/:id
func (sh *subscriptionHandler) GetSubscription(c echo.Context) error {
	id := c.Param("id")

	subscription, err := sh.Payment.GetSubscription(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Subscription not found"})
	}

	return c.JSON(http.StatusOK, subscription)
}

// UpdateSubscription handles PUT /subscriptions/:id
func (sh *subscriptionHandler) UpdateSubscription(c echo.Context) error {
	id := c.Param("id")

	var subscription models.Subscription
	if err := c.Bind(&subscription); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}
	subscription.ID = id

	if err := sh.Payment.UpdateSubscription(&subscription); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update subscription"})
	}

	return c.NoContent(http.StatusOK)
}

// CancelSubscription handles POST /subscriptions/:id/cancel
func (sh *subscriptionHandler) CancelSubscription(c echo.Context) error {
	id := c.Param("id")
	var req struct {
		CancelAtPeriodEnd bool `json:"cancel_at_period_end"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	if err := sh.Payment.CancelSubscription(id, req.CancelAtPeriodEnd); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to cancel subscription"})
	}

	return c.NoContent(http.StatusOK)
}

// ListSubscriptions handles GET /subscriptions
func (sh *subscriptionHandler) ListSubscriptions(c echo.Context) error {
	customerID := c.QueryParam("customer_id")

	subscriptions, err := sh.Payment.ListSubscriptions(c.Request().Context(), customerID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list subscriptions"})
	}

	return c.JSON(http.StatusOK, subscriptions)
}
