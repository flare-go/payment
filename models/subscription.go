package models

import (
	"goflare.io/payment/models/enum"
	"time"
)

// Subscription 代表客戶的訂閱
// Subscription represents a customer's subscription
type Subscription struct {
	ID                 uint64                  `json:"id"`
	CustomerID         uint64                  `json:"customer_id"`
	PriceID            uint64                  `json:"price_id"`
	Status             enum.SubscriptionStatus `json:"status"`
	CurrentPeriodStart time.Time               `json:"current_period_start"`
	CurrentPeriodEnd   time.Time               `json:"current_period_end"`
	CanceledAt         *time.Time              `json:"canceled_at,omitempty"`
	CancelAtPeriodEnd  bool                    `json:"cancel_at_period_end"`
	TrialStart         *time.Time              `json:"trial_start,omitempty"`
	TrialEnd           *time.Time              `json:"trial_end,omitempty"`
	StripeID           string                  `json:"stripe_id"`
	CreatedAt          time.Time               `json:"created_at"`
	UpdatedAt          time.Time               `json:"updated_at"`
}
