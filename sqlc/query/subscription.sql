-- name: CreateSubscription :exec
INSERT INTO subscriptions (
    id,
    customer_id,
    price_id,
    status,
    current_period_start,
    current_period_end,
    cancel_at_period_end,
    trial_start,
    trial_end
) VALUES (
             $1, $2, $3, $4, $5, $6, $7, $8, $9
         );
-- RETURNING id, customer_id, price_id, status, current_period_start, current_period_end, canceled_at, cancel_at_period_end, trial_start, trial_end, stripe_id, created_at, updated_at;

-- name: GetSubscription :one
SELECT id, customer_id, price_id, status, current_period_start, current_period_end, canceled_at, cancel_at_period_end, trial_start, trial_end, created_at, updated_at
FROM subscriptions
WHERE id = $1;

-- name: UpdateSubscription :exec
UPDATE subscriptions
SET price_id = $2,
    status = $3,
    current_period_start = $4,
    current_period_end = $5,
    canceled_at = $6,
    cancel_at_period_end = $7,
    trial_start = $8,
    trial_end = $9,
    updated_at = NOW()
WHERE id = $1;
-- RETURNING id, customer_id, price_id, status, current_period_start, current_period_end, canceled_at, cancel_at_period_end, trial_start, trial_end, stripe_id, created_at, updated_at;

-- name: CancelSubscription :exec
UPDATE subscriptions
SET status = 'canceled',
    canceled_at = NOW(),
    updated_at = NOW()
WHERE id = $1;
-- RETURNING id, customer_id, price_id, status, current_period_start, current_period_end, canceled_at, cancel_at_period_end, trial_start, trial_end, stripe_id, created_at, updated_at;

-- name: ListSubscriptions :many
SELECT id, customer_id, price_id, status, current_period_start, current_period_end, canceled_at, cancel_at_period_end, trial_start, trial_end, created_at, updated_at
FROM subscriptions
WHERE customer_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: DeleteSubscription :exec
DELETE FROM subscriptions
WHERE id = $1;

-- name: GetExpiringSubscriptions :many
SELECT id, customer_id, price_id, status, current_period_start, current_period_end, canceled_at, cancel_at_period_end, trial_start, trial_end, created_at, updated_at
FROM subscriptions
WHERE current_period_end <= $1 AND status = $2
ORDER BY current_period_end;

-- name: UpsertSubscription :exec
INSERT INTO subscriptions (
    id, customer_id, price_id, status, current_period_start, current_period_end,
    canceled_at, cancel_at_period_end, trial_start, trial_end
) VALUES (
             $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
         )
ON CONFLICT (id) DO UPDATE SET
                                      customer_id = EXCLUDED.customer_id,
                                      price_id = EXCLUDED.price_id,
                                      status = EXCLUDED.status,
                                      current_period_start = EXCLUDED.current_period_start,
                                      current_period_end = EXCLUDED.current_period_end,
                                      canceled_at = EXCLUDED.canceled_at,
                                      cancel_at_period_end = EXCLUDED.cancel_at_period_end,
                                      trial_start = EXCLUDED.trial_start,
                                      trial_end = EXCLUDED.trial_end,
                                      updated_at = NOW();