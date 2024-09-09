-- name: CreatePrice :exec
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
         );
-- RETURNING id, product_id, type, currency, unit_amount, recurring_interval, recurring_interval_count, trial_period_days, active, stripe_id, created_at, updated_at;

-- name: GetPrice :one
SELECT id, product_id, type, currency, unit_amount, recurring_interval, recurring_interval_count, trial_period_days, active, created_at, updated_at
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
    active = $9,
    updated_at = NOW()
WHERE id = $1;
-- RETURNING id, product_id, type, currency, unit_amount, recurring_interval, recurring_interval_count, trial_period_days, active, stripe_id, created_at, updated_at;

-- name: DeletePrice :one
UPDATE prices
SET active = false
WHERE id = $1
RETURNING product_id;

-- name: ListPrices :many
SELECT id, product_id, type, currency, unit_amount, recurring_interval, recurring_interval_count, trial_period_days, active, created_at, updated_at
FROM prices
WHERE product_id = $1
ORDER BY created_at DESC;

-- name: ListActivePrices :many
SELECT id, product_id, type, currency, unit_amount, recurring_interval, recurring_interval_count, trial_period_days, active, created_at, updated_at
FROM prices
WHERE product_id = $1 AND active = true
ORDER BY created_at DESC;


-- name: UpsertPrice :exec
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
                                      updated_at = NOW();