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
    id, name, amount_off, percent_off, currency, duration,
    duration_in_months, max_redemptions, times_redeemed, valid, redeem_by
) VALUES (
             $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
         )
ON CONFLICT (id)
    DO UPDATE SET
                  name = EXCLUDED.name,
                  amount_off = EXCLUDED.amount_off,
                  percent_off = EXCLUDED.percent_off,
                  currency = EXCLUDED.currency,
                  duration = EXCLUDED.duration,
                  duration_in_months = EXCLUDED.duration_in_months,
                  max_redemptions = EXCLUDED.max_redemptions,
                  times_redeemed = EXCLUDED.times_redeemed,
                  valid = EXCLUDED.valid,
                  redeem_by = EXCLUDED.redeem_by,
                  updated_at = NOW();
