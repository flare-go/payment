// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: price.sql

package sqlc

import (
	"context"
)

const createPrice = `-- name: CreatePrice :exec
INSERT INTO prices (
    id,
    product_id,
    type,
    currency,
    unit_amount,
    recurring_interval,
    recurring_interval_count,
    trial_period_days,
    active
) VALUES (
             $1, $2, $3, $4, $5, $6, $7, $8, true
         )
`

type CreatePriceParams struct {
	ID                     string                     `json:"id"`
	ProductID              string                     `json:"productId"`
	Type                   PriceType                  `json:"type"`
	Currency               Currency                   `json:"currency"`
	UnitAmount             float64                    `json:"unitAmount"`
	RecurringInterval      NullPriceRecurringInterval `json:"recurringInterval"`
	RecurringIntervalCount int32                      `json:"recurringIntervalCount"`
	TrialPeriodDays        int32                      `json:"trialPeriodDays"`
}

func (q *Queries) CreatePrice(ctx context.Context, arg CreatePriceParams) error {
	_, err := q.db.Exec(ctx, createPrice,
		arg.ID,
		arg.ProductID,
		arg.Type,
		arg.Currency,
		arg.UnitAmount,
		arg.RecurringInterval,
		arg.RecurringIntervalCount,
		arg.TrialPeriodDays,
	)
	return err
}

const deletePrice = `-- name: DeletePrice :one

UPDATE prices
SET active = false
WHERE id = $1
RETURNING product_id
`

// RETURNING id, product_id, type, currency, unit_amount, recurring_interval, recurring_interval_count, trial_period_days, active, stripe_id, created_at, updated_at;
func (q *Queries) DeletePrice(ctx context.Context, id string) (string, error) {
	row := q.db.QueryRow(ctx, deletePrice, id)
	var product_id string
	err := row.Scan(&product_id)
	return product_id, err
}

const getPrice = `-- name: GetPrice :one

SELECT id, product_id, type, currency, unit_amount, recurring_interval, recurring_interval_count, trial_period_days, active, created_at, updated_at
FROM prices
WHERE id = $1 LIMIT 1
`

// RETURNING id, product_id, type, currency, unit_amount, recurring_interval, recurring_interval_count, trial_period_days, active, stripe_id, created_at, updated_at;
func (q *Queries) GetPrice(ctx context.Context, id string) (*Price, error) {
	row := q.db.QueryRow(ctx, getPrice, id)
	var i Price
	err := row.Scan(
		&i.ID,
		&i.ProductID,
		&i.Type,
		&i.Currency,
		&i.UnitAmount,
		&i.RecurringInterval,
		&i.RecurringIntervalCount,
		&i.TrialPeriodDays,
		&i.Active,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return &i, err
}

const listActivePrices = `-- name: ListActivePrices :many
SELECT id, product_id, type, currency, unit_amount, recurring_interval, recurring_interval_count, trial_period_days, active, created_at, updated_at
FROM prices
WHERE product_id = $1 AND active = true
ORDER BY created_at DESC
`

func (q *Queries) ListActivePrices(ctx context.Context, productID string) ([]*Price, error) {
	rows, err := q.db.Query(ctx, listActivePrices, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []*Price{}
	for rows.Next() {
		var i Price
		if err := rows.Scan(
			&i.ID,
			&i.ProductID,
			&i.Type,
			&i.Currency,
			&i.UnitAmount,
			&i.RecurringInterval,
			&i.RecurringIntervalCount,
			&i.TrialPeriodDays,
			&i.Active,
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

const listPrices = `-- name: ListPrices :many
SELECT id, product_id, type, currency, unit_amount, recurring_interval, recurring_interval_count, trial_period_days, active, created_at, updated_at
FROM prices
WHERE product_id = $1
ORDER BY created_at DESC
`

func (q *Queries) ListPrices(ctx context.Context, productID string) ([]*Price, error) {
	rows, err := q.db.Query(ctx, listPrices, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []*Price{}
	for rows.Next() {
		var i Price
		if err := rows.Scan(
			&i.ID,
			&i.ProductID,
			&i.Type,
			&i.Currency,
			&i.UnitAmount,
			&i.RecurringInterval,
			&i.RecurringIntervalCount,
			&i.TrialPeriodDays,
			&i.Active,
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

const updatePrice = `-- name: UpdatePrice :exec
UPDATE prices
SET product_id = $2,
    type = $3,
    currency = $4,
    unit_amount = $5,
    recurring_interval = $6,
    recurring_interval_count = $7,
    trial_period_days = $8,
    active = $9,
    updated_at = NOW()
WHERE id = $1
`

type UpdatePriceParams struct {
	ID                     string                     `json:"id"`
	ProductID              string                     `json:"productId"`
	Type                   PriceType                  `json:"type"`
	Currency               Currency                   `json:"currency"`
	UnitAmount             float64                    `json:"unitAmount"`
	RecurringInterval      NullPriceRecurringInterval `json:"recurringInterval"`
	RecurringIntervalCount int32                      `json:"recurringIntervalCount"`
	TrialPeriodDays        int32                      `json:"trialPeriodDays"`
	Active                 bool                       `json:"active"`
}

func (q *Queries) UpdatePrice(ctx context.Context, arg UpdatePriceParams) error {
	_, err := q.db.Exec(ctx, updatePrice,
		arg.ID,
		arg.ProductID,
		arg.Type,
		arg.Currency,
		arg.UnitAmount,
		arg.RecurringInterval,
		arg.RecurringIntervalCount,
		arg.TrialPeriodDays,
		arg.Active,
	)
	return err
}

const upsertPrice = `-- name: UpsertPrice :exec
INSERT INTO prices (
    id, product_id, type, currency, unit_amount, recurring_interval, recurring_interval_count,
    trial_period_days, active
) VALUES (
             $1, $2, $3, $4, $5, $6, $7, $8, $9
         )
ON CONFLICT (id) DO UPDATE SET
                                      product_id = EXCLUDED.product_id,
                                      type = EXCLUDED.type,
                                      currency = EXCLUDED.currency,
                                      unit_amount = EXCLUDED.unit_amount,
                                      recurring_interval = EXCLUDED.recurring_interval,
                                      recurring_interval_count = EXCLUDED.recurring_interval_count,
                                      trial_period_days = EXCLUDED.trial_period_days,
                                      active = EXCLUDED.active,
                                      updated_at = NOW()
`

type UpsertPriceParams struct {
	ID                     string                     `json:"id"`
	ProductID              string                     `json:"productId"`
	Type                   PriceType                  `json:"type"`
	Currency               Currency                   `json:"currency"`
	UnitAmount             float64                    `json:"unitAmount"`
	RecurringInterval      NullPriceRecurringInterval `json:"recurringInterval"`
	RecurringIntervalCount int32                      `json:"recurringIntervalCount"`
	TrialPeriodDays        int32                      `json:"trialPeriodDays"`
	Active                 bool                       `json:"active"`
}

func (q *Queries) UpsertPrice(ctx context.Context, arg UpsertPriceParams) error {
	_, err := q.db.Exec(ctx, upsertPrice,
		arg.ID,
		arg.ProductID,
		arg.Type,
		arg.Currency,
		arg.UnitAmount,
		arg.RecurringInterval,
		arg.RecurringIntervalCount,
		arg.TrialPeriodDays,
		arg.Active,
	)
	return err
}
