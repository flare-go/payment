package models

import (
	"time"

	"goflare.io/payment/models/enum"
	"goflare.io/payment/sqlc"
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

func NewSubscription() *Subscription {
	return &Subscription{}
}

func (s *Subscription) ConvertFromSQLCSubscription(sqlcSubscription any) *Subscription {

	var (
		id, customerID, priceID uint64
		stripeID                string
		cancelAtPeriodEnd       bool
		currentPeriodStart,
		currentPeriodEnd,
		canceledAt,
		trialStart,
		trialEnd,
		createdAt,
		updatedAt time.Time
	)

	switch sp := sqlcSubscription.(type) {
	case *sqlc.Subscription:
		id = sp.ID
		customerID = sp.CustomerID
		priceID = sp.PriceID
		stripeID = sp.StripeID
		cancelAtPeriodEnd = sp.CancelAtPeriodEnd
		currentPeriodStart = sp.CurrentPeriodStart.Time
		currentPeriodEnd = sp.CurrentPeriodEnd.Time
		canceledAt = sp.CanceledAt.Time
		trialStart = sp.TrialStart.Time
		trialEnd = sp.TrialEnd.Time
		createdAt = sp.CreatedAt.Time
		updatedAt = sp.UpdatedAt.Time
	default:
		return nil
	}

	s.ID = id
	s.CustomerID = customerID
	s.PriceID = priceID
	s.StripeID = stripeID
	s.CancelAtPeriodEnd = cancelAtPeriodEnd
	s.CurrentPeriodStart = currentPeriodStart
	s.CurrentPeriodEnd = currentPeriodEnd
	s.CanceledAt = &canceledAt
	s.TrialStart = &trialStart
	s.TrialEnd = &trialEnd
	s.CreatedAt = createdAt
	s.UpdatedAt = updatedAt

	return s
}
