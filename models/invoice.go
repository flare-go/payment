package models

import (
	"goflare.io/payment/models/enum"
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
	AmountDue       int64              `json:"amount_due"`
	AmountPaid      int64              `json:"amount_paid"`
	AmountRemaining int64              `json:"amount_remaining"`
	DueDate         time.Time          `json:"due_date"`
	PaidAt          *time.Time         `json:"paid_at,omitempty"`
	StripeID        string             `json:"stripe_id"`
	InvoiceItems    []InvoiceItem      `json:"invoice_items"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

type InvoiceItem struct {
	ID          uint64 `json:"id"`
	InvoiceID   uint64 `json:"invoice_id"`
	Amount      int64  `json:"amount"`
	Description string `json:"description"`
}
