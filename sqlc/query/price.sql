-- name: CreatePrice :exec
INSERT INTO prices (
    product_id,
    type,
    currency,
    unit_amount,
    recurring_interval,
    recurring_interval_count,
    trial_period_days,
    active,
    stripe_id
) VALUES (
             $1, $2, $3, $4, $5, $6, $7, true, $8
         );
-- RETURNING id, product_id, type, currency, unit_amount, recurring_interval, recurring_interval_count, trial_period_days, active, stripe_id, created_at, updated_at;

-- name: GetPrice :one
SELECT id, product_id, type, currency, unit_amount, recurring_interval, recurring_interval_count, trial_period_days, active, stripe_id, created_at, updated_at
FROM prices
WHERE id = $1 LIMIT 1;

-- name: UpdatePrice :exec
UPDATE prices
SET product_id = $2,
    type = $3,
    currency = $4,
    unit_amount = $5,
    recurring_interval = $6,
    recurring_interval_count = $7,
    trial_period_days = $8,
    stripe_id = $9,
    active = $10,
    updated_at = NOW()
WHERE id = $1;
-- RETURNING id, product_id, type, currency, unit_amount, recurring_interval, recurring_interval_count, trial_period_days, active, stripe_id, created_at, updated_at;

-- name: DeletePrice :one
UPDATE prices
SET active = false
WHERE id = $1
RETURNING product_id;

-- name: ListPrices :many
SELECT id, product_id, type, currency, unit_amount, recurring_interval, recurring_interval_count, trial_period_days, active, stripe_id, created_at, updated_at
FROM prices
WHERE product_id = $1
ORDER BY created_at DESC;

-- name: ListActivePrices :many
SELECT id, product_id, type, currency, unit_amount, recurring_interval, recurring_interval_count, trial_period_days, active, stripe_id, created_at, updated_at
FROM prices
WHERE product_id = $1 AND active = true
ORDER BY created_at DESC;