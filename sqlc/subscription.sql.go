// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: subscription.sql

package sqlc

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const cancelSubscription = `-- name: CancelSubscription :exec

UPDATE subscriptions
SET status = 'CANCELED',
    canceled_at = NOW(),
    updated_at = NOW()
WHERE id = $1
`

// RETURNING id, customer_id, price_id, status, current_period_start, current_period_end, canceled_at, cancel_at_period_end, trial_start, trial_end, stripe_id, created_at, updated_at;
func (q *Queries) CancelSubscription(ctx context.Context, id uint64) error {
	_, err := q.db.Exec(ctx, cancelSubscription, id)
	return err
}

const createSubscription = `-- name: CreateSubscription :exec
INSERT INTO subscriptions (
    customer_id,
    price_id,
    status,
    current_period_start,
    current_period_end,
    cancel_at_period_end,
    trial_start,
    trial_end,
    stripe_id
) VALUES (
             $1, $2, $3, $4, $5, $6, $7, $8, $9
         )
`

type CreateSubscriptionParams struct {
	CustomerID         uint64             `json:"customerId"`
	PriceID            uint64             `json:"priceId"`
	Status             SubscriptionStatus `json:"status"`
	CurrentPeriodStart pgtype.Timestamptz `json:"currentPeriodStart"`
	CurrentPeriodEnd   pgtype.Timestamptz `json:"currentPeriodEnd"`
	CancelAtPeriodEnd  bool               `json:"cancelAtPeriodEnd"`
	TrialStart         pgtype.Timestamptz `json:"trialStart"`
	TrialEnd           pgtype.Timestamptz `json:"trialEnd"`
	StripeID           string             `json:"stripeId"`
}

func (q *Queries) CreateSubscription(ctx context.Context, arg CreateSubscriptionParams) error {
	_, err := q.db.Exec(ctx, createSubscription,
		arg.CustomerID,
		arg.PriceID,
		arg.Status,
		arg.CurrentPeriodStart,
		arg.CurrentPeriodEnd,
		arg.CancelAtPeriodEnd,
		arg.TrialStart,
		arg.TrialEnd,
		arg.StripeID,
	)
	return err
}

const getExpiringSubscriptions = `-- name: GetExpiringSubscriptions :many
SELECT id, customer_id, price_id, status, current_period_start, current_period_end, canceled_at, cancel_at_period_end, trial_start, trial_end, stripe_id, created_at, updated_at
FROM subscriptions
WHERE current_period_end <= $1 AND status = $2
ORDER BY current_period_end
`

type GetExpiringSubscriptionsParams struct {
	CurrentPeriodEnd pgtype.Timestamptz `json:"currentPeriodEnd"`
	Status           SubscriptionStatus `json:"status"`
}

func (q *Queries) GetExpiringSubscriptions(ctx context.Context, arg GetExpiringSubscriptionsParams) ([]*Subscription, error) {
	rows, err := q.db.Query(ctx, getExpiringSubscriptions, arg.CurrentPeriodEnd, arg.Status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []*Subscription{}
	for rows.Next() {
		var i Subscription
		if err := rows.Scan(
			&i.ID,
			&i.CustomerID,
			&i.PriceID,
			&i.Status,
			&i.CurrentPeriodStart,
			&i.CurrentPeriodEnd,
			&i.CanceledAt,
			&i.CancelAtPeriodEnd,
			&i.TrialStart,
			&i.TrialEnd,
			&i.StripeID,
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

const getSubscription = `-- name: GetSubscription :one

SELECT id, customer_id, price_id, status, current_period_start, current_period_end, canceled_at, cancel_at_period_end, trial_start, trial_end, stripe_id, created_at, updated_at
FROM subscriptions
WHERE id = $1 LIMIT 1
`

// RETURNING id, customer_id, price_id, status, current_period_start, current_period_end, canceled_at, cancel_at_period_end, trial_start, trial_end, stripe_id, created_at, updated_at;
func (q *Queries) GetSubscription(ctx context.Context, id uint64) (*Subscription, error) {
	row := q.db.QueryRow(ctx, getSubscription, id)
	var i Subscription
	err := row.Scan(
		&i.ID,
		&i.CustomerID,
		&i.PriceID,
		&i.Status,
		&i.CurrentPeriodStart,
		&i.CurrentPeriodEnd,
		&i.CanceledAt,
		&i.CancelAtPeriodEnd,
		&i.TrialStart,
		&i.TrialEnd,
		&i.StripeID,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return &i, err
}

const listSubscriptions = `-- name: ListSubscriptions :many

SELECT id, customer_id, price_id, status, current_period_start, current_period_end, canceled_at, cancel_at_period_end, trial_start, trial_end, stripe_id, created_at, updated_at
FROM subscriptions
WHERE customer_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3
`

type ListSubscriptionsParams struct {
	CustomerID uint64 `json:"customerId"`
	Limit      int64  `json:"limit"`
	Offset     int64  `json:"offset"`
}

// RETURNING id, customer_id, price_id, status, current_period_start, current_period_end, canceled_at, cancel_at_period_end, trial_start, trial_end, stripe_id, created_at, updated_at;
func (q *Queries) ListSubscriptions(ctx context.Context, arg ListSubscriptionsParams) ([]*Subscription, error) {
	rows, err := q.db.Query(ctx, listSubscriptions, arg.CustomerID, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []*Subscription{}
	for rows.Next() {
		var i Subscription
		if err := rows.Scan(
			&i.ID,
			&i.CustomerID,
			&i.PriceID,
			&i.Status,
			&i.CurrentPeriodStart,
			&i.CurrentPeriodEnd,
			&i.CanceledAt,
			&i.CancelAtPeriodEnd,
			&i.TrialStart,
			&i.TrialEnd,
			&i.StripeID,
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

const listSubscriptionsByStripeID = `-- name: ListSubscriptionsByStripeID :many
SELECT id, customer_id, price_id, status, current_period_start, current_period_end, canceled_at, cancel_at_period_end, trial_start, trial_end, stripe_id, created_at, updated_at
FROM subscriptions
WHERE stripe_id = $1
`

func (q *Queries) ListSubscriptionsByStripeID(ctx context.Context, stripeID string) ([]*Subscription, error) {
	rows, err := q.db.Query(ctx, listSubscriptionsByStripeID, stripeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []*Subscription{}
	for rows.Next() {
		var i Subscription
		if err := rows.Scan(
			&i.ID,
			&i.CustomerID,
			&i.PriceID,
			&i.Status,
			&i.CurrentPeriodStart,
			&i.CurrentPeriodEnd,
			&i.CanceledAt,
			&i.CancelAtPeriodEnd,
			&i.TrialStart,
			&i.TrialEnd,
			&i.StripeID,
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

const updateSubscription = `-- name: UpdateSubscription :exec
UPDATE subscriptions
SET price_id = $2,
    status = $3,
    current_period_start = $4,
    current_period_end = $5,
    canceled_at = $6,
    cancel_at_period_end = $7,
    trial_start = $8,
    trial_end = $9,
    stripe_id = $10,
    updated_at = NOW()
WHERE id = $1
`

type UpdateSubscriptionParams struct {
	ID                 uint64             `json:"id"`
	PriceID            uint64             `json:"priceId"`
	Status             SubscriptionStatus `json:"status"`
	CurrentPeriodStart pgtype.Timestamptz `json:"currentPeriodStart"`
	CurrentPeriodEnd   pgtype.Timestamptz `json:"currentPeriodEnd"`
	CanceledAt         pgtype.Timestamptz `json:"canceledAt"`
	CancelAtPeriodEnd  bool               `json:"cancelAtPeriodEnd"`
	TrialStart         pgtype.Timestamptz `json:"trialStart"`
	TrialEnd           pgtype.Timestamptz `json:"trialEnd"`
	StripeID           string             `json:"stripeId"`
}

func (q *Queries) UpdateSubscription(ctx context.Context, arg UpdateSubscriptionParams) error {
	_, err := q.db.Exec(ctx, updateSubscription,
		arg.ID,
		arg.PriceID,
		arg.Status,
		arg.CurrentPeriodStart,
		arg.CurrentPeriodEnd,
		arg.CanceledAt,
		arg.CancelAtPeriodEnd,
		arg.TrialStart,
		arg.TrialEnd,
		arg.StripeID,
	)
	return err
}
