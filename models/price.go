package models

import (
	"goflare.io/payment/models/enum"
	"goflare.io/payment/sqlc"
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

func NewPrice() *Price {
	return &Price{}
}

func (p *Price) ConvertFromSQLCPrice(sqlcPrice any) *Price {

	var (
		stripeID                                string
		id, productID                           uint64
		recurringIntervalCount, trialPeriodDays int32
		unitAmount                              float64
		active                                  bool
		currency                                enum.Currency
		priceType                               enum.PriceType
		recurringInterval                       enum.Interval
		createdAt, updatedAt                    time.Time
	)

	switch sp := sqlcPrice.(type) {
	case *sqlc.Price:
		id = sp.ID
		productID = sp.ProductID
		stripeID = sp.StripeID
		recurringIntervalCount = sp.RecurringIntervalCount
		trialPeriodDays = sp.TrialPeriodDays
		unitAmount = sp.UnitAmount
		active = sp.Active
		currency = enum.Currency(sp.Currency)
		priceType = enum.PriceType(sp.Type)
		recurringInterval = enum.Interval(sp.RecurringInterval.IntervalType)
		createdAt = sp.CreatedAt.Time
		updatedAt = sp.UpdatedAt.Time
	default:
		return nil
	}

	p.ID = id
	p.ProductID = productID
	p.StripeID = stripeID
	p.Active = active
	p.RecurringIntervalCount = recurringIntervalCount
	p.TrialPeriodDays = trialPeriodDays
	p.UnitAmount = unitAmount
	p.Active = active
	p.Currency = currency
	p.Type = priceType
	p.RecurringInterval = recurringInterval
	p.CreatedAt = createdAt
	p.UpdatedAt = updatedAt

	return p
}
