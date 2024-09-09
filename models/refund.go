package models

import (
	"goflare.io/payment/models/enum"
	"goflare.io/payment/sqlc"
	"time"
)

type Refund struct {
	ID        string            `json:"id"`
	ChargeID  string            `json:"charge_id"`
	Amount    float64           `json:"amount"`
	Status    enum.RefundStatus `json:"status"`
	Reason    string            `json:"reason"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type PartialRefund struct {
	ID        string
	ChargeID  *string
	Amount    *float64
	Status    *enum.RefundStatus
	Reason    *string
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

func NewRefund() *Refund {
	return &Refund{}
}

func (r *Refund) ConvertFromSQLCRefund(sqlcRefund any) *Refund {

	var (
		amount               float64
		id, chargeID, reason string
		status               enum.RefundStatus
		createdAt, updatedAt time.Time
	)

	switch sp := sqlcRefund.(type) {
	case *sqlc.Refund:
		id = sp.ID
		chargeID = sp.ChargeID
		amount = sp.Amount
		if sp.Reason != nil {
			reason = *sp.Reason
		}
		status = enum.RefundStatus(sp.Status)
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
