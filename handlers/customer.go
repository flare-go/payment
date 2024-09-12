package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"goflare.io/payment"
	"goflare.io/payment/models"
)

type CustomerHandler interface {
	CreateCustomer(c echo.Context) error
	GetCustomer(c echo.Context) error
	UpdateCustomer(c echo.Context) error
	DeleteCustomer(c echo.Context) error
}

type customerHandler struct {
	Payment payment.Payment
}

func NewCustomerHandler(
	Payment payment.Payment,
) CustomerHandler {
	return &customerHandler{
		Payment: Payment,
	}
}

// CreateCustomer handles POST /customers
func (ch *customerHandler) CreateCustomer(c echo.Context) error {
	var customer models.Customer
	if err := c.Bind(&customer); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	if err := ch.Payment.CreateCustomer(c.Request().Context(), customer.Email, customer.Name); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create customer"})
	}

	return c.NoContent(http.StatusCreated)
}

// GetCustomer handles GET /customers/:id
func (ch *customerHandler) GetCustomer(c echo.Context) error {
	id := c.Param("id")

	customer, err := ch.Payment.GetCustomer(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Customer not found"})
	}

	return c.JSON(http.StatusOK, customer)
}

// UpdateCustomer handles PUT /customers/:id
func (ch *customerHandler) UpdateCustomer(c echo.Context) error {
	id := c.Param("id")

	var customer models.Customer
	if err := c.Bind(&customer); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}
	customer.ID = id

	if err := ch.Payment.UpdateCustomerBalance(&customer); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update customer"})
	}

	return c.NoContent(http.StatusOK)
}

// DeleteCustomer handles DELETE /customers/:id
func (ch *customerHandler) DeleteCustomer(c echo.Context) error {
	id := c.Param("id")

	if err := ch.Payment.DeleteCustomer(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete customer"})
	}

	return c.NoContent(http.StatusNoContent)
}

// ListCustomers handles GET /customers
//func (ch *customerHandler) ListCustomers(c echo.Context) error {
//	limit, _ := strconv.ParseUint(c.QueryParam("limit"), 10, 64)
//	offset, _ := strconv.ParseUint(c.QueryParam("offset"), 10, 64)
//
//	if limit == 0 {
//		limit = 10 // 默認限制
//	}
//
//	customers, err := ch.Payment.List(c.Request().Context(), limit, offset)
//	if err != nil {
//		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list customers"})
//	}
//
//	return c.JSON(http.StatusOK, customers)
//}
