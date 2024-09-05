package models

import (
	"goflare.io/payment/models/enum"
	"goflare.io/payment/sqlc"
	"time"
)

type Refund struct {
	ID              uint64            `json:"id"`
	PaymentIntentID uint64            `json:"payment_intent_id"`
	Amount          float64           `json:"amount"`
	Status          enum.RefundStatus `json:"status"`
	Reason          string            `json:"reason"`
	StripeID        string            `json:"stripe_id"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

func NewRefund() *Refund {
	return &Refund{}
}

func (r *Refund) ConvertFromSQLCRefund(sqlcRefund any) *Refund {

	var (
		id, paymentIntentID  uint64
		amount               float64
		reason, stripeID     string
		status               enum.RefundStatus
		createdAt, updatedAt time.Time
	)

	switch sp := sqlcRefund.(type) {
	case *sqlc.Refund:
		id = sp.ID
		paymentIntentID = sp.PaymentIntentID
		amount = sp.Amount
		if sp.Reason != nil {
			reason = *sp.Reason
		}
		status = enum.RefundStatus(sp.Status)
		stripeID = sp.StripeID
		createdAt = sp.CreatedAt.Time
		updatedAt = sp.UpdatedAt.Time
	default:
		return nil
	}

	r.ID = id
	r.PaymentIntentID = paymentIntentID
	r.Amount = amount
	r.Reason = reason
	r.StripeID = stripeID
	r.Status = status
	r.CreatedAt = createdAt
	r.UpdatedAt = updatedAt

	return r
}
