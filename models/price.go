package models

import (
	"goflare.io/payment/models/enum"
	"time"
)

// Price 代表產品的價格方案
// Price represents a pricing plan for a product
type Price struct {
	ID                     uint64         `json:"id"`
	ProductID              uint64         `json:"product_id"`
	Type                   enum.PriceType `json:"type"`
	Currency               enum.Currency  `json:"currency"`
	UnitAmount             float64        `json:"unit_amount"`
	RecurringInterval      enum.Interval  `json:"recurring_interval,omitempty"`
	RecurringIntervalCount int32          `json:"recurring_interval_count,omitempty"`
	TrialPeriodDays        int32          `json:"trial_period_days,omitempty"`
	Active                 bool           `json:"active"`
	StripeID               string         `json:"stripe_id"`
	CreatedAt              time.Time      `json:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at"`
}
