package models

import (
	"github.com/stripe/stripe-go/v79"
	"time"

	"goflare.io/payment/sqlc"
)

// Price 代表產品的價格方案
// Price represents a pricing plan for a product
type Price struct {
	ID                     string                        `json:"id"`
	ProductID              string                        `json:"product_id"`
	Type                   stripe.PriceType              `json:"type"`
	Currency               stripe.Currency               `json:"currency"`
	UnitAmount             float64                       `json:"unit_amount"`
	RecurringInterval      stripe.PriceRecurringInterval `json:"recurring_interval,omitempty"`
	RecurringIntervalCount int32                         `json:"recurring_interval_count,omitempty"`
	TrialPeriodDays        int32                         `json:"trial_period_days,omitempty"`
	Active                 bool                          `json:"active"`
	CreatedAt              time.Time                     `json:"created_at"`
	UpdatedAt              time.Time                     `json:"updated_at"`
}
type PartialPrice struct {
	ID                     string
	ProductID              *string
	Active                 *bool
	Currency               *stripe.Currency
	UnitAmount             *float64
	Type                   *stripe.PriceType
	RecurringInterval      *stripe.PriceRecurringInterval
	RecurringIntervalCount *int32
	TrialPeriodDays        *int32
	CreatedAt              *time.Time
	UpdatedAt              *time.Time
}

func NewPrice() *Price {
	return &Price{}
}

func (p *Price) ConvertFromSQLCPrice(sqlcPrice any) *Price {

	var (
		id, productID                           string
		recurringIntervalCount, trialPeriodDays int32
		unitAmount                              float64
		active                                  bool
		currency                                stripe.Currency
		priceType                               stripe.PriceType
		recurringInterval                       stripe.PriceRecurringInterval
		createdAt, updatedAt                    time.Time
	)

	switch sp := sqlcPrice.(type) {
	case *sqlc.Price:
		id = sp.ID
		productID = sp.ProductID
		recurringIntervalCount = sp.RecurringIntervalCount
		trialPeriodDays = sp.TrialPeriodDays
		unitAmount = sp.UnitAmount
		active = sp.Active
		currency = stripe.Currency(sp.Currency)
		priceType = stripe.PriceType(sp.Type)
		if sp.RecurringInterval.Valid {
			recurringInterval = stripe.PriceRecurringInterval(sp.RecurringInterval.PriceRecurringInterval)
		}
		createdAt = sp.CreatedAt.Time
		updatedAt = sp.UpdatedAt.Time
	default:
		return nil
	}

	p.ID = id
	p.ProductID = productID
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
