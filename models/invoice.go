package models

import (
	"time"

	"github.com/stripe/stripe-go/v79"

	"goflare.io/payment/sqlc"
)

// Invoice 代表訂閱或一次性購買的發票
// Invoice represents an invoice for a subscription or one-time purchase
type Invoice struct {
	ID              string               `json:"id"`
	CustomerID      string               `json:"customer_id"`
	SubscriptionID  *string              `json:"subscription_id,omitempty"`
	Status          stripe.InvoiceStatus `json:"status"`
	Currency        stripe.Currency      `json:"currency"`
	AmountDue       float64              `json:"amount_due"`
	AmountPaid      float64              `json:"amount_paid"`
	AmountRemaining float64              `json:"amount_remaining"`
	InvoiceItems    []*InvoiceItem       `json:"invoice_items"`
	DueDate         time.Time            `json:"due_date"`
	PaidAt          time.Time            `json:"paid_at"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
}

type PartialInvoice struct {
	ID              string
	CustomerID      *string
	SubscriptionID  *string
	Status          *stripe.InvoiceStatus
	Currency        *stripe.Currency
	AmountDue       *float64
	AmountPaid      *float64
	AmountRemaining *float64
	DueDate         *time.Time
	PaidAt          *time.Time
	CreatedAt       *time.Time
	UpdatedAt       *time.Time
}

func NewInvoice() *Invoice {
	return &Invoice{}
}

func (i *Invoice) ConvertFromSQLCInvoice(sqlcInvoice any) *Invoice {

	var (
		id, customerID, subscriptionID         string
		amountDue, amountPaid, amountRemaining float64
		status                                 stripe.InvoiceStatus
		currency                               stripe.Currency
		dueDate, paidAt, createdAt, updatedAt  time.Time
	)

	switch sp := sqlcInvoice.(type) {
	case *sqlc.Invoice:
		if sp.SubscriptionID != nil {
			subscriptionID = *sp.SubscriptionID
		}
		id = sp.ID
		customerID = sp.CustomerID
		amountDue = sp.AmountDue
		amountPaid = sp.AmountPaid
		amountRemaining = sp.AmountRemaining
		currency = stripe.Currency(sp.Currency)
		status = stripe.InvoiceStatus(sp.Status)
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
	i.DueDate = dueDate
	i.PaidAt = paidAt
	i.CreatedAt = createdAt
	i.UpdatedAt = updatedAt

	return i
}
