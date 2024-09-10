package models

import "time"

type Review struct {
	ID              string     `json:"id"`
	PaymentIntentID string     `json:"payment_intent_id"`
	Reason          string     `json:"reason"`
	Status          string     `json:"status"`
	OpenedAt        time.Time  `json:"opened_at"`
	ClosedAt        *time.Time `json:"closed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type PartialReview struct {
	ID              string     `json:"id"`
	PaymentIntentID *string    `json:"payment_intent_id,omitempty"`
	Reason          *string    `json:"reason"`
	ClosedReason    *string    `json:"close_reason,omitempty"`
	Status          *string    `json:"status,omitempty"`
	OpenedAt        *time.Time `json:"opened_at,omitempty"`
	ClosedAt        *time.Time `json:"closed_at,omitempty"`
	CreatedAt       *time.Time `json:"created_at,omitempty"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
}
