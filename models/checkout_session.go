package models

import "time"

type CheckoutSession struct {
	ID              string    `json:"id"`
	CustomerID      *string   `json:"customer_id,omitempty"`
	PaymentIntentID *string   `json:"payment_intent_id,omitempty"`
	Status          string    `json:"status"`
	Mode            string    `json:"mode"`
	SuccessURL      string    `json:"success_url"`
	CancelURL       string    `json:"cancel_url"`
	AmountTotal     int64     `json:"amount_total"`
	Currency        string    `json:"currency"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type PartialCheckoutSession struct {
	ID              string     `json:"id"`
	CustomerID      *string    `json:"customer_id,omitempty"`
	PaymentIntentID *string    `json:"payment_intent_id,omitempty"`
	Status          *string    `json:"status,omitempty"`
	Mode            *string    `json:"mode,omitempty"`
	SuccessURL      *string    `json:"success_url,omitempty"`
	CancelURL       *string    `json:"cancel_url,omitempty"`
	AmountTotal     *int64     `json:"amount_total,omitempty"`
	Currency        *string    `json:"currency,omitempty"`
	CreatedAt       *time.Time `json:"created_at,omitempty"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
}
