package models

import (
	"github.com/stripe/stripe-go/v79"
	"time"
)

type PaymentLink struct {
	ID        string          `json:"id"`
	Active    bool            `json:"active"`
	URL       string          `json:"url"`
	Amount    int64           `json:"amount"`
	Currency  stripe.Currency `json:"currency"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type PartialPaymentLink struct {
	ID        string           `json:"id"`
	Active    *bool            `json:"active,omitempty"`
	URL       *string          `json:"url,omitempty"`
	Amount    *int64           `json:"amount,omitempty"`
	Currency  *stripe.Currency `json:"currency,omitempty"`
	CreatedAt *time.Time       `json:"created_at,omitempty"`
	UpdatedAt *time.Time       `json:"updated_at,omitempty"`
}
