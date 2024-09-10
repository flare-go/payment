package models

import (
	"github.com/stripe/stripe-go/v79"
	"goflare.io/payment/sqlc"
	"time"
)

type Dispute struct {
	ID            string               `json:"id"`
	ChargeID      string               `json:"charge_id"`
	Amount        int64                `json:"amount"`
	Currency      stripe.Currency      `json:"currency"`
	Status        stripe.DisputeStatus `json:"status"`
	Reason        stripe.DisputeReason `json:"reason"`
	EvidenceDueBy time.Time            `json:"evidence_due_by"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
}

type PartialDispute struct {
	ID            string
	ChargeID      *string
	Currency      *stripe.Currency
	Amount        *int64
	Status        *stripe.DisputeStatus
	Reason        *stripe.DisputeReason
	EvidenceDueBy *time.Time
	CreatedAt     *time.Time
	UpdatedAt     *time.Time
}

func NewDispute() *Dispute {
	return &Dispute{}
}

func (d *Dispute) ConvertFromSQLCDispute(sqlcDispute any) *Dispute {

	var (
		id, chargeID                        string
		status                              stripe.DisputeStatus
		reason                              stripe.DisputeReason
		currency                            stripe.Currency
		evidenceDueBy, createdAt, updatedAt time.Time
	)

	switch sp := sqlcDispute.(type) {
	case *sqlc.Dispute:
		id = sp.ID
		chargeID = sp.ChargeID
		status = stripe.DisputeStatus(sp.Status)
		reason = stripe.DisputeReason(sp.Reason)
		currency = stripe.Currency(sp.Currency)
		evidenceDueBy = sp.EvidenceDueBy.Time
		createdAt = sp.CreatedAt.Time
		updatedAt = sp.UpdatedAt.Time
	default:
		return nil
	}

	d.ID = id
	d.ChargeID = chargeID
	d.Status = status
	d.Reason = reason
	d.EvidenceDueBy = evidenceDueBy
	d.Currency = currency
	d.CreatedAt = createdAt
	d.UpdatedAt = updatedAt

	return d
}
