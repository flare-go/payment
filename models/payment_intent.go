package models

import (
	"time"

	"github.com/stripe/stripe-go/v79"

	"goflare.io/payment/sqlc"
)

// PaymentIntent 代表一次支付意圖
// PaymentIntent represents a payment intent
type PaymentIntent struct {
	ID               string
	CustomerID       string
	Amount           float64
	Currency         stripe.Currency
	Status           stripe.PaymentIntentStatus
	PaymentMethodID  string
	SetupFutureUsage stripe.PaymentIntentSetupFutureUsage
	ClientSecret     string
	CaptureMethod    stripe.PaymentIntentCaptureMethod
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type PartialPaymentIntent struct {
	ID               string
	CustomerID       *string
	Amount           *float64
	Currency         *stripe.Currency
	Status           *stripe.PaymentIntentStatus
	PaymentMethodID  *string
	SetupFutureUsage *stripe.PaymentIntentSetupFutureUsage
	ClientSecret     *string
	CaptureMethod    *stripe.PaymentIntentCaptureMethod
	CreatedAt        *time.Time
	UpdatedAt        *time.Time
}

func NewPaymentIntent() *PaymentIntent {
	return &PaymentIntent{}
}

func (pi *PaymentIntent) ConvertFromSQLCPaymentIntent(sqlcPaymentIntent any) *PaymentIntent {

	var (
		id, customerID, clientSecret, paymentMethodID string
		amount                                        float64
		captureMethod                                 stripe.PaymentIntentCaptureMethod
		setupFutureUsage                              stripe.PaymentIntentSetupFutureUsage
		currency                                      stripe.Currency
		status                                        stripe.PaymentIntentStatus
		createdAt, updatedAt                          time.Time
	)

	switch sp := sqlcPaymentIntent.(type) {
	case *sqlc.PaymentIntent:
		id = sp.ID
		customerID = sp.CustomerID
		amount = sp.Amount
		currency = stripe.Currency(sp.Currency)
		status = stripe.PaymentIntentStatus(sp.Status)
		if sp.PaymentMethodID != nil {
			paymentMethodID = *sp.PaymentMethodID
		}
		if sp.SetupFutureUsage.Valid {
			setupFutureUsage = stripe.PaymentIntentSetupFutureUsage(sp.SetupFutureUsage.PaymentIntentSetupFutureUsage)
		}
		captureMethod = stripe.PaymentIntentCaptureMethod(sp.CaptureMethod)
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
	pi.CaptureMethod = captureMethod
	pi.CreatedAt = createdAt
	pi.UpdatedAt = updatedAt

	return pi
}
