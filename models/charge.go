package models

import "time"

type Charge struct {
	ID              string    `json:"id"`
	CustomerID      string    `json:"customer_id"`
	PaymentIntentID string    `json:"payment_intent_id"`
	Amount          float64   `json:"amount"`
	Currency        string    `json:"currency"`
	Status          string    `json:"status"`
	Paid            bool      `json:"paid"`
	Refunded        bool      `json:"refunded"`
	FailureCode     string    `json:"failure_code"`
	FailureMessage  string    `json:"failure_message"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type PartialCharge struct {
	ID              string
	CustomerID      *string
	PaymentIntentID *string
	Amount          *float64
	Currency        *string
	Status          *string
	Paid            *bool
	Refunded        *bool
	FailureCode     *string
	FailureMessage  *string
	CreatedAt       *time.Time
	UpdatedAt       *time.Time
}
