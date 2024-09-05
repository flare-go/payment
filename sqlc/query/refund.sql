-- name: CreateRefund :exec
INSERT INTO refunds (
    payment_intent_id,
    amount,
    status,
    reason,
    stripe_id
) VALUES (
             $1, $2, $3, $4, $5
         );

-- name: GetRefund :one
SELECT
    id,
    payment_intent_id,
    amount,
    status,
    reason,
    stripe_id,
    created_at,
    updated_at
FROM refunds
WHERE id = $1;

-- name: UpdateRefund :exec
UPDATE refunds
SET
    status = $2,
    reason = $3,
    updated_at = NOW()
WHERE id = $1;

-- name: ListRefunds :many
SELECT
    id,
    payment_intent_id,
    amount,
    status,
    reason,
    stripe_id,
    created_at,
    updated_at
FROM refunds
WHERE payment_intent_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListRefundsByStripeID :many
SELECT
    id,
    payment_intent_id,
    amount,
    status,
    reason,
    stripe_id,
    created_at,
    updated_at
FROM refunds
WHERE stripe_id = $1
ORDER BY created_at DESC;

-- name: ListByPaymentIntentID :many
SELECT
    id,
    payment_intent_id,
    amount,
    status,
    reason,
    stripe_id,
    created_at,
    updated_at
FROM refunds
WHERE payment_intent_id = $1
ORDER BY created_at DESC;

-- name: DeleteRefund :exec
DELETE FROM refunds
WHERE id = $1;