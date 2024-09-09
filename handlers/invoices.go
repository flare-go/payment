package handlers

import (
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"goflare.io/payment"
	"net/http"
)

type InvoiceHandler interface {
	GetInvoice(c echo.Context) error
	ListInvoices(c echo.Context) error
	PayInvoice(c echo.Context) error
	//CreateDraftInvoice(c echo.Context) error // 用於特殊情況
}

type invoiceHandler struct {
	Payment payment.Payment
	logger  *zap.Logger
}

func NewInvoiceHandler(
	Payment payment.Payment,
	logger *zap.Logger,
) InvoiceHandler {
	return &invoiceHandler{
		Payment: Payment,
		logger:  logger,
	}
}

// GetInvoice handles GET /invoices/:id
func (ih *invoiceHandler) GetInvoice(c echo.Context) error {
	id := c.Param("id")
	invoice, err := ih.Payment.GetInvoice(c.Request().Context(), id)
	if err != nil {
		ih.logger.Error("Failed to get invoice", zap.Error(err), zap.String("id", id))
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Invoice not found"})
	}

	return c.JSON(http.StatusOK, invoice)
}

// ListInvoices handles GET /invoices
func (ih *invoiceHandler) ListInvoices(c echo.Context) error {
	customerID := c.QueryParam("customer_id")

	invoices, err := ih.Payment.ListInvoices(c.Request().Context(), customerID)
	if err != nil {
		ih.logger.Error("Failed to list invoices", zap.Error(err), zap.String("customerID", customerID))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list invoices"})
	}

	return c.JSON(http.StatusOK, invoices)
}

// PayInvoice handles POST /invoices/:id/pay
func (ih *invoiceHandler) PayInvoice(c echo.Context) error {
	id := c.Param("id")

	if err := ih.Payment.PayInvoice(id); err != nil {
		ih.logger.Error("Failed to pay invoice", zap.Error(err), zap.String("id", id))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to pay invoice"})
	}

	return c.NoContent(http.StatusOK)
}

// CreateDraftInvoice handles POST /invoices/draft
//func (ih *invoiceHandler) CreateDraftInvoice(c echo.Context) error {
//	var req struct {
//		CustomerID uint64 `json:"customer_id"`
//		// 其他需要的字段
//	}
//	if err := c.Bind(&req); err != nil {
//		ih.logger.Error("Invalid request payload", zap.Error(err))
//		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
//	}
//
//	draftInvoice, err := ih.Payment.CreateDraftInvoice(c.Request().Context(), req.CustomerID)
//	if err != nil {
//		ih.logger.Error("Failed to create draft invoice", zap.Error(err), zap.Uint64("customerID", req.CustomerID))
//		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create draft invoice"})
//	}
//
//	return c.JSON(http.StatusCreated, draftInvoice)
//}
