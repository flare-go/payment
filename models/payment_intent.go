package models

import (
	"goflare.io/payment/models/enum"
	"time"
)

// PaymentIntent 代表一次支付意圖
// PaymentIntent represents a payment intent
type PaymentIntent struct {
	ID               uint64
	CustomerID       uint64
	Amount           uint64
	Currency         enum.Currency
	Status           enum.PaymentIntentStatus
	PaymentMethodID  *uint64
	SetupFutureUsage *string
	StripeID         string
	ClientSecret     string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
