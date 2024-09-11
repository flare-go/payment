package models

import (
	"time"

	"github.com/stripe/stripe-go/v79"

	"goflare.io/payment/sqlc"
)

type Refund struct {
	ID        string              `json:"id"`
	ChargeID  string              `json:"charge_id"`
	Amount    float64             `json:"amount"`
	Status    stripe.RefundStatus `json:"status"`
	Reason    stripe.RefundReason `json:"reason"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
}

type PartialRefund struct {
	ID        string
	ChargeID  *string
	Amount    *float64
	Status    *stripe.RefundStatus
	Reason    *stripe.RefundReason
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

func NewRefund() *Refund {
	return &Refund{}
}

func (r *Refund) ConvertFromSQLCRefund(sqlcRefund any) *Refund {

	var (
		amount               float64
		id, chargeID         string
		reason               stripe.RefundReason
		status               stripe.RefundStatus
		createdAt, updatedAt time.Time
	)

	switch sp := sqlcRefund.(type) {
	case *sqlc.Refund:
		id = sp.ID
		chargeID = sp.ChargeID
		amount = sp.Amount
		if sp.Reason.Valid {
			reason = stripe.RefundReason(sp.Reason.RefundReason)
		}
		status = stripe.RefundStatus(sp.Status)
		createdAt = sp.CreatedAt.Time
		updatedAt = sp.UpdatedAt.Time
	default:
		return nil
	}

	r.ID = id
	r.ChargeID = chargeID
	r.Amount = amount
	r.Reason = reason
	r.Status = status
	r.CreatedAt = createdAt
	r.UpdatedAt = updatedAt

	return r
}
