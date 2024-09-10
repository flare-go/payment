package models

import (
	"github.com/stripe/stripe-go/v79"
	"time"
)

type Quote struct {
	ID          string             `json:"id"`
	CustomerID  string             `json:"customer_id"`
	Status      stripe.QuoteStatus `json:"status"`
	AmountTotal int64              `json:"amount_total"`
	Currency    stripe.Currency    `json:"currency"`
	ValidUntil  *time.Time         `json:"valid_until,omitempty"`
	AcceptedAt  *time.Time         `json:"accepted_at,omitempty"`
	CanceledAt  *time.Time         `json:"canceled_at,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

type PartialQuote struct {
	ID          string              `json:"id"`
	CustomerID  *string             `json:"customer_id,omitempty"`
	Status      *stripe.QuoteStatus `json:"status,omitempty"`
	AmountTotal *int64              `json:"amount_total,omitempty"`
	Currency    *stripe.Currency    `json:"currency,omitempty"`
	ValidUntil  *time.Time          `json:"valid_until,omitempty"`
	AcceptedAt  *time.Time          `json:"accepted_at,omitempty"`
	CanceledAt  *time.Time          `json:"canceled_at,omitempty"`
	CreatedAt   *time.Time          `json:"created_at,omitempty"`
	UpdatedAt   *time.Time          `json:"updated_at,omitempty"`
}
