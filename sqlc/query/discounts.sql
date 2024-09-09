-- name: CreateDiscount :one
INSERT INTO discounts (
    id, customer_id, coupon_id, start, "end"
) VALUES (
             $1, $2, $3, $4, $5
         )
RETURNING *;

-- name: GetDiscountByID :one
SELECT * FROM discounts
WHERE id = $1 LIMIT 1;

-- name: ListDiscounts :many
SELECT * FROM discounts
ORDER BY id
LIMIT $1 OFFSET $2;

-- name: ListDiscountsByCustomerID :many
SELECT * FROM discounts
WHERE customer_id = $1
ORDER BY id
LIMIT $2 OFFSET $3;

-- name: UpdateDiscount :one
UPDATE discounts
SET customer_id = $2,
    coupon_id = $3,
    start = $4,
    "end" = $5,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteDiscount :exec
DELETE FROM discounts
WHERE id = $1;

-- name: UpsertDiscount :exec
INSERT INTO discounts (
    id, customer_id, coupon_id, start, "end"
) VALUES (
             $1, $2, $3, $4, $5
         )
ON CONFLICT (id)
    DO UPDATE SET
                  customer_id = EXCLUDED.customer_id,
                  coupon_id = EXCLUDED.coupon_id,
                  start = EXCLUDED.start,
                  "end" = EXCLUDED."end",
                  updated_at = NOW();