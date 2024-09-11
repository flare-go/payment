package models

import (
	"time"

	"github.com/stripe/stripe-go/v79"
)

type Review struct {
	ID              string                    `json:"id"`
	PaymentIntentID string                    `json:"payment_intent_id"`
	Reason          stripe.ReviewReason       `json:"reason"`
	ClosedReason    stripe.ReviewClosedReason `json:"close_reason"`
	Status          string                    `json:"status"`
	OpenedAt        time.Time                 `json:"opened_at"`
	ClosedAt        *time.Time                `json:"closed_at,omitempty"`
	CreatedAt       time.Time                 `json:"created_at"`
	UpdatedAt       time.Time                 `json:"updated_at"`
}

type PartialReview struct {
	ID              string                     `json:"id"`
	PaymentIntentID *string                    `json:"payment_intent_id,omitempty"`
	Reason          *stripe.ReviewReason       `json:"reason"`
	ClosedReason    *stripe.ReviewClosedReason `json:"close_reason,omitempty"`
	Status          *string                    `json:"status,omitempty"`
	OpenedAt        *time.Time                 `json:"opened_at,omitempty"`
	ClosedAt        *time.Time                 `json:"closed_at,omitempty"`
	CreatedAt       *time.Time                 `json:"created_at,omitempty"`
	UpdatedAt       *time.Time                 `json:"updated_at,omitempty"`
}
