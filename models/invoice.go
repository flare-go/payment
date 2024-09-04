package models

import (
	"goflare.io/payment/models/enum"
	"goflare.io/payment/sqlc"
	"time"
)

// Invoice 代表訂閱或一次性購買的發票
// Invoice represents an invoice for a subscription or one-time purchase
type Invoice struct {
	ID              uint64             `json:"id"`
	CustomerID      uint64             `json:"customer_id"`
	SubscriptionID  *uint64            `json:"subscription_id,omitempty"`
	Status          enum.InvoiceStatus `json:"status"`
	Currency        enum.Currency      `json:"currency"`
	AmountDue       uint64             `json:"amount_due"`
	AmountPaid      uint64             `json:"amount_paid"`
	AmountRemaining uint64             `json:"amount_remaining"`
	StripeID        string             `json:"stripe_id"`
	InvoiceItems    []*InvoiceItem     `json:"invoice_items"`
	DueDate         time.Time          `json:"due_date"`
	PaidAt          time.Time          `json:"paid_at"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

func NewInvoice() *Invoice {
	return &Invoice{}
}

func (i *Invoice) ConvertFromSQLCInvoice(sqlcInvoice any) *Invoice {

	var (
		stripeID                               string
		id, customerID, subscriptionID         uint64
		status                                 enum.InvoiceStatus
		currency                               enum.Currency
		amountDue, amountPaid, amountRemaining int64
		dueDate, paidAt, createdAt, updatedAt  time.Time
	)

	switch sp := sqlcInvoice.(type) {
	case *sqlc.Invoice:
		id = sp.ID
		customerID = sp.CustomerID
		subscriptionID = sp.SubscriptionID
		amountDue = sp.AmountDue
		amountPaid = sp.AmountPaid
		amountRemaining = sp.AmountRemaining
		currency = enum.Currency(sp.Currency)
		stripeID = sp.StripeID
		status = enum.InvoiceStatus(sp.Status)
		dueDate = sp.DueDate.Time
		paidAt = sp.PaidAt.Time
		createdAt = sp.CreatedAt.Time
		updatedAt = sp.UpdatedAt.Time
	default:
		return nil
	}

	i.ID = id
	i.CustomerID = customerID
	i.SubscriptionID = &subscriptionID
	i.AmountDue = amountDue
	i.AmountPaid = amountPaid
	i.AmountRemaining = amountRemaining
	i.Currency = currency
	i.Status = status
	i.StripeID = stripeID
	i.DueDate = dueDate
	i.PaidAt = paidAt
	i.CreatedAt = createdAt
	i.UpdatedAt = updatedAt

	return i
}
