// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: refund.sql

package sqlc

import (
	"context"
)

const createRefund = `-- name: CreateRefund :exec
INSERT INTO refunds (
    id,
    charge_id,
    amount,
    status,
    reason
) VALUES (
             $1, $2, $3, $4, $5
         )
`

type CreateRefundParams struct {
	ID       string           `json:"id"`
	ChargeID string           `json:"chargeId"`
	Amount   float64          `json:"amount"`
	Status   RefundStatus     `json:"status"`
	Reason   NullRefundReason `json:"reason"`
}

func (q *Queries) CreateRefund(ctx context.Context, arg CreateRefundParams) error {
	_, err := q.db.Exec(ctx, createRefund,
		arg.ID,
		arg.ChargeID,
		arg.Amount,
		arg.Status,
		arg.Reason,
	)
	return err
}

const deleteRefund = `-- name: DeleteRefund :exec
DELETE FROM refunds
WHERE id = $1
`

func (q *Queries) DeleteRefund(ctx context.Context, id string) error {
	_, err := q.db.Exec(ctx, deleteRefund, id)
	return err
}

const getRefund = `-- name: GetRefund :one
SELECT
    id,
    charge_id,
    amount,
    status,
    reason,
    created_at,
    updated_at
FROM refunds
WHERE id = $1
`

func (q *Queries) GetRefund(ctx context.Context, id string) (*Refund, error) {
	row := q.db.QueryRow(ctx, getRefund, id)
	var i Refund
	err := row.Scan(
		&i.ID,
		&i.ChargeID,
		&i.Amount,
		&i.Status,
		&i.Reason,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return &i, err
}

const listByChargeID = `-- name: ListByChargeID :many
SELECT
    id,
    charge_id,
    amount,
    status,
    reason,
    created_at,
    updated_at
FROM refunds
WHERE charge_id = $1
ORDER BY created_at DESC
`

func (q *Queries) ListByChargeID(ctx context.Context, chargeID string) ([]*Refund, error) {
	rows, err := q.db.Query(ctx, listByChargeID, chargeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []*Refund{}
	for rows.Next() {
		var i Refund
		if err := rows.Scan(
			&i.ID,
			&i.ChargeID,
			&i.Amount,
			&i.Status,
			&i.Reason,
			&i.CreatedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listRefunds = `-- name: ListRefunds :many
SELECT
    id,
    charge_id,
    amount,
    status,
    reason,
    created_at,
    updated_at
FROM refunds
WHERE charge_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3
`

type ListRefundsParams struct {
	ChargeID string `json:"chargeId"`
	Limit    int64  `json:"limit"`
	Offset   int64  `json:"offset"`
}

func (q *Queries) ListRefunds(ctx context.Context, arg ListRefundsParams) ([]*Refund, error) {
	rows, err := q.db.Query(ctx, listRefunds, arg.ChargeID, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []*Refund{}
	for rows.Next() {
		var i Refund
		if err := rows.Scan(
			&i.ID,
			&i.ChargeID,
			&i.Amount,
			&i.Status,
			&i.Reason,
			&i.CreatedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const updateRefund = `-- name: UpdateRefund :exec
UPDATE refunds
SET
    status = $2,
    reason = $3,
    updated_at = NOW()
WHERE id = $1
`

type UpdateRefundParams struct {
	ID     string           `json:"id"`
	Status RefundStatus     `json:"status"`
	Reason NullRefundReason `json:"reason"`
}

func (q *Queries) UpdateRefund(ctx context.Context, arg UpdateRefundParams) error {
	_, err := q.db.Exec(ctx, updateRefund, arg.ID, arg.Status, arg.Reason)
	return err
}

const upsertRefund = `-- name: UpsertRefund :exec
INSERT INTO refunds (
    id, charge_id, amount, status, reason
) VALUES (
             $1, $2, $3, $4, $5
         )
ON CONFLICT (id) DO UPDATE SET
                                      charge_id = EXCLUDED.charge_id,
                                      amount = EXCLUDED.amount,
                                      status = EXCLUDED.status,
                                      reason = EXCLUDED.reason,
                                      updated_at = NOW()
`

type UpsertRefundParams struct {
	ID       string           `json:"id"`
	ChargeID string           `json:"chargeId"`
	Amount   float64          `json:"amount"`
	Status   RefundStatus     `json:"status"`
	Reason   NullRefundReason `json:"reason"`
}

func (q *Queries) UpsertRefund(ctx context.Context, arg UpsertRefundParams) error {
	_, err := q.db.Exec(ctx, upsertRefund,
		arg.ID,
		arg.ChargeID,
		arg.Amount,
		arg.Status,
		arg.Reason,
	)
	return err
}
