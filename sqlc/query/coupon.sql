-- name: CreateCoupon :one
INSERT INTO coupons (
    id, name, amount_off, percent_off, currency, duration,
    duration_in_months, max_redemptions, times_redeemed, valid, redeem_by
) VALUES (
             $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
         )
RETURNING *;

-- name: GetCouponByID :one
SELECT * FROM coupons WHERE id = $1 LIMIT 1;

-- name: ListCoupons :many
SELECT * FROM coupons
ORDER BY id
LIMIT $1 OFFSET $2;

-- name: UpdateCoupon :one
UPDATE coupons
SET name = $2,
    amount_off = $3,
    percent_off = $4,
    currency = $5,
    duration = $6,
    duration_in_months = $7,
    max_redemptions = $8,
    times_redeemed = $9,
    valid = $10,
    redeem_by = $11,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteCoupon :exec
DELETE FROM coupons WHERE id = $1;

-- name: UpsertCoupon :exec
INSERT INTO coupons (
    id,
    name,
    currency,
    duration,
    amount_off,
    percent_off,
    duration_in_months,
    max_redemptions,
    times_redeemed,
    valid,
    redeem_by,
    created_at,
    updated_at
) VALUES (
             $1,
             sqlc.narg('name'),
             sqlc.narg('currency'),
             sqlc.narg('duration'),
             sqlc.narg('amount_off'),
             sqlc.narg('percent_off'),
             sqlc.narg('duration_in_months'),
             sqlc.narg('max_redemptions'),
             $2,
             $3,
             $4,
             $5,
             $6
         )
ON CONFLICT (id) DO UPDATE SET
                               name = COALESCE(sqlc.narg('name'), coupons.name),
                               currency = COALESCE(sqlc.narg('currency'), coupons.currency),
                               duration = COALESCE(sqlc.narg('duration'), coupons.duration),
                               amount_off = COALESCE(sqlc.narg('amount_off'), coupons.amount_off),
                               percent_off = COALESCE(sqlc.narg('percent_off'), coupons.percent_off),
                               duration_in_months = COALESCE(sqlc.narg('duration_in_months'), coupons.duration_in_months),
                               max_redemptions = COALESCE(sqlc.narg('max_redemptions'), coupons.max_redemptions),
                               times_redeemed = $2,
                               valid = $3,
                               redeem_by = $4,
                               updated_at = $6;