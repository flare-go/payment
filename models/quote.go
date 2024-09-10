package models

import "time"

type Quote struct {
	ID          string     `json:"id"`
	CustomerID  string     `json:"customer_id"`
	Status      string     `json:"status"`
	AmountTotal int64      `json:"amount_total"`
	Currency    string     `json:"currency"`
	ValidUntil  *time.Time `json:"valid_until,omitempty"`
	AcceptedAt  *time.Time `json:"accepted_at,omitempty"`
	CanceledAt  *time.Time `json:"canceled_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type PartialQuote struct {
	ID          string     `json:"id"`
	CustomerID  *string    `json:"customer_id,omitempty"`
	Status      *string    `json:"status,omitempty"`
	AmountTotal *int64     `json:"amount_total,omitempty"`
	Currency    *string    `json:"currency,omitempty"`
	ValidUntil  *time.Time `json:"valid_until,omitempty"`
	AcceptedAt  *time.Time `json:"accepted_at,omitempty"`
	CanceledAt  *time.Time `json:"canceled_at,omitempty"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}
