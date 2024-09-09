package models

import (
	"time"

	"goflare.io/payment/models/enum"
	"goflare.io/payment/sqlc"
)

// PaymentIntent 代表一次支付意圖
// PaymentIntent represents a payment intent
type PaymentIntent struct {
	ID               string
	CustomerID       string
	Amount           float64
	Currency         enum.Currency
	Status           enum.PaymentIntentStatus
	PaymentMethodID  string
	SetupFutureUsage string
	ClientSecret     string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type PartialPaymentIntent struct {
	ID               string
	CustomerID       *string
	Amount           *float64
	Currency         *enum.Currency
	Status           *enum.PaymentIntentStatus
	PaymentMethodID  *string
	SetupFutureUsage *string
	ClientSecret     *string
	CaptureMethod    *string
	CreatedAt        *time.Time
	UpdatedAt        *time.Time
}

func NewPaymentIntent() *PaymentIntent {
	return &PaymentIntent{}
}

func (pi *PaymentIntent) ConvertFromSQLCPaymentIntent(sqlcPaymentIntent any) *PaymentIntent {

	var (
		id, customerID, clientSecret, setupFutureUsage, paymentMethodID string
		amount                                                          float64
		currency                                                        enum.Currency
		status                                                          enum.PaymentIntentStatus
		createdAt, updatedAt                                            time.Time
	)

	switch sp := sqlcPaymentIntent.(type) {
	case *sqlc.PaymentIntent:
		id = sp.ID
		customerID = sp.CustomerID
		amount = sp.Amount
		currency = enum.Currency(sp.Currency)
		status = enum.PaymentIntentStatus(sp.Status)
		if sp.PaymentMethodID != nil {
			paymentMethodID = *sp.PaymentMethodID
		}
		if sp.SetupFutureUsage != nil {
			setupFutureUsage = *sp.SetupFutureUsage
		}
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
	pi.ClientSecret = clientSecret
	pi.CreatedAt = createdAt
	pi.UpdatedAt = updatedAt

	return pi
}
