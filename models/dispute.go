package models

import (
	"goflare.io/payment/models/enum"
	"goflare.io/payment/sqlc"
	"time"
)

type Dispute struct {
	ID            string        `json:"id"`
	ChargeID      string        `json:"charge_id"`
	Amount        int64         `json:"amount"`
	Currency      enum.Currency `json:"currency"`
	Status        string        `json:"status"`
	Reason        string        `json:"reason"`
	EvidenceDueBy time.Time     `json:"evidence_due_by"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

type PartialDispute struct {
	ID        string
	ChargeID  *string
	Amount    *int64
	Status    *string
	Reason    *string
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

func NewDispute() *Dispute {
	return &Dispute{}
}

func (d *Dispute) ConvertFromSQLCDispute(sqlcDispute any) *Dispute {

	var (
		id, chargeID, status, reason        string
		currency                            enum.Currency
		evidenceDueBy, createdAt, updatedAt time.Time
	)

	switch sp := sqlcDispute.(type) {
	case *sqlc.Dispute:
		id = sp.ID
		chargeID = sp.ChargeID
		status = sp.Status
		reason = sp.Reason
		currency = enum.Currency(sp.Currency)
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
