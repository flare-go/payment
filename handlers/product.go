package handlers

import (
	"errors"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"goflare.io/payment"
	"goflare.io/payment/models"
	"net/http"
)

type ProductHandler interface {
	CreateProduct(c echo.Context) error
	GetProduct(c echo.Context) error
	UpdateProduct(c echo.Context) error
	DeleteProduct(c echo.Context) error
	ListProducts(c echo.Context) error
}

type productHandler struct {
	Payment payment.Payment
	Logger  *zap.Logger
}

func NewProductHandler(payment payment.Payment, logger *zap.Logger) ProductHandler {
	return &productHandler{
		Payment: payment,
		Logger:  logger,
	}
}

func (ph *productHandler) CreateProduct(c echo.Context) error {

	var req models.Product
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	// 驗證請求數據
	if err := validateCreateProductRequest(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if err := ph.Payment.CreateProduct(req); err != nil {
		ph.Logger.Error("Failed to create product", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create product"})
	}

	return c.NoContent(http.StatusCreated)
}

func (ph *productHandler) GetProduct(c echo.Context) error {
	ctx := c.Request().Context()

	id := c.Param("id")

	product, err := ph.Payment.GetProductWithAllPrices(ctx, id)
	if err != nil {
		ph.Logger.Error("Failed to get product", zap.Error(err), zap.String("id", id))
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Product not found"})
	}

	return c.JSON(http.StatusOK, product)
}

func (ph *productHandler) UpdateProduct(c echo.Context) error {

	id := c.Param("id")

	var productReq struct {
		Name        string            `json:"name"`
		Description string            `json:"description"`
		Active      bool              `json:"active"`
		Metadata    map[string]string `json:"metadata"`
		StripeID    string            `json:"stripe_id"`
	}

	if err := c.Bind(&productReq); err != nil {
		ph.Logger.Error("Failed to bind product request", zap.Error(err))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	product := &models.Product{
		ID:          id,
		Name:        productReq.Name,
		Description: productReq.Description,
		Active:      productReq.Active,
		Metadata:    productReq.Metadata,
	}

	if err := ph.Payment.UpdateProduct(product); err != nil {
		ph.Logger.Error("Failed to update product", zap.Error(err), zap.String("id", id))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update product"})
	}

	return c.NoContent(http.StatusOK)
}

func (ph *productHandler) DeleteProduct(c echo.Context) error {

	id := c.Param("id")

	if err := ph.Payment.DeleteProduct(id); err != nil {
		ph.Logger.Error("Failed to delete product", zap.Error(err), zap.String("id", id))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete product"})
	}

	return c.NoContent(http.StatusNoContent)
}

func (ph *productHandler) ListProducts(c echo.Context) error {
	ctx := c.Request().Context()

	products, err := ph.Payment.ListProducts(ctx)
	if err != nil {
		ph.Logger.Error("Failed to list products", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list products"})
	}

	return c.JSON(http.StatusOK, products)
}

func validateCreateProductRequest(req models.Product) error {
	if len(req.Name) < 2 {
		return errors.New("product name must be at least 2 characters long")
	}
	if len(req.Prices) == 0 {
		return errors.New("at least one price must be provided")
	}
	// 添加更多驗證邏輯
	return nil
}
