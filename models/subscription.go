package models

import (
	"github.com/stripe/stripe-go/v79"
	"time"

	"goflare.io/payment/sqlc"
)

// Subscription 代表客戶的訂閱
// Subscription represents a customer's subscription
type Subscription struct {
	ID                 string                    `json:"id"`
	CustomerID         string                    `json:"customer_id"`
	PriceID            string                    `json:"price_id"`
	Status             stripe.SubscriptionStatus `json:"status"`
	CurrentPeriodStart time.Time                 `json:"current_period_start"`
	CurrentPeriodEnd   time.Time                 `json:"current_period_end"`
	CanceledAt         *time.Time                `json:"canceled_at,omitempty"`
	CancelAtPeriodEnd  bool                      `json:"cancel_at_period_end"`
	TrialStart         *time.Time                `json:"trial_start,omitempty"`
	TrialEnd           *time.Time                `json:"trial_end,omitempty"`
	CreatedAt          time.Time                 `json:"created_at"`
	UpdatedAt          time.Time                 `json:"updated_at"`
}

type PartialSubscription struct {
	ID                 string
	CustomerID         *string
	PriceID            *string
	Status             *stripe.SubscriptionStatus
	CurrentPeriodStart *time.Time
	CurrentPeriodEnd   *time.Time
	CanceledAt         *time.Time
	CancelAtPeriodEnd  *bool
	TrialStart         *time.Time
	TrialEnd           *time.Time
	CreatedAt          *time.Time
	UpdatedAt          *time.Time
}

func NewSubscription() *Subscription {
	return &Subscription{}
}

func (s *Subscription) ConvertFromSQLCSubscription(sqlcSubscription any) *Subscription {

	var (
		id, customerID, priceID string
		status                  stripe.SubscriptionStatus
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
		status = stripe.SubscriptionStatus(sp.Status)
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
	s.Status = status
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
