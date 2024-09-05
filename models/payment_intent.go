package models

import (
	"time"

	"goflare.io/payment/models/enum"
	"goflare.io/payment/sqlc"
)

// PaymentIntent 代表一次支付意圖
// PaymentIntent represents a payment intent
type PaymentIntent struct {
	ID               uint64
	CustomerID       uint64
	Amount           float64
	Currency         enum.Currency
	Status           enum.PaymentIntentStatus
	PaymentMethodID  *uint64
	SetupFutureUsage *string
	StripeID         string
	ClientSecret     string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func NewPaymentIntent() *PaymentIntent {
	return &PaymentIntent{}
}

func (pi *PaymentIntent) ConvertFromSQLCPaymentIntent(sqlcPaymentIntent any) *PaymentIntent {

	var (
		id, customerID         uint64
		amount                 float64
		currency               enum.Currency
		status                 enum.PaymentIntentStatus
		paymentMethodID        *uint64
		setupFutureUsage       *string
		stripeID, clientSecret string
		createdAt, updatedAt   time.Time
	)

	switch sp := sqlcPaymentIntent.(type) {
	case *sqlc.PaymentIntent:
		id = sp.ID
		customerID = sp.CustomerID
		amount = sp.Amount
		currency = enum.Currency(sp.Currency)
		status = enum.PaymentIntentStatus(sp.Status)
		paymentMethodID = &sp.PaymentMethodID
		setupFutureUsage = &sp.SetupFutureUsage
		stripeID = sp.StripeID
		clientSecret = sp.ClientSecret
		createdAt = sp.CreatedAt.Time
		updatedAt = sp.UpdatedAt.Time
	default:
		return nil
	}

	pi.ID = id
	pi.CustomerID = customerID
	pi.Amount = amount
	pi.Currency = currency
	pi.Status = status
	pi.PaymentMethodID = paymentMethodID
	pi.SetupFutureUsage = setupFutureUsage
	pi.StripeID = stripeID
	pi.ClientSecret = clientSecret
	pi.CreatedAt = createdAt
	pi.UpdatedAt = updatedAt

	return pi
}
