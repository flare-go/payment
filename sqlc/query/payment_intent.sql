-- name: CreatePaymentIntent :exec
INSERT INTO payment_intents (
    customer_id,
    amount,
    currency,
    status,
    payment_method_id,
    setup_future_usage,
    stripe_id,
    client_secret
) VALUES (
             $1, $2, $3, $4, $5, $6, $7, $8
         );
-- RETURNING id, customer_id, amount, currency, status, payment_method_id, setup_future_usage, stripe_id, client_secret, created_at, updated_at;

-- name: GetPaymentIntent :one
SELECT id, customer_id, amount, currency, status, payment_method_id, setup_future_usage, stripe_id, client_secret, created_at, updated_at
FROM payment_intents
WHERE id = $1 LIMIT 1;

-- name: UpdatePaymentIntent :exec
UPDATE payment_intents
SET status = $2,
    payment_method_id = $3,
    setup_future_usage = $4,
    stripe_id = $5,
    client_secret = $6,
    updated_at = NOW()
WHERE id = $1;
-- RETURNING id, customer_id, amount, currency, status, payment_method_id, setup_future_usage, stripe_id, client_secret, created_at, updated_at;

-- name: ListPaymentIntents :many
SELECT id, customer_id, amount, currency, status, payment_method_id, setup_future_usage, stripe_id, client_secret, created_at, updated_at
FROM payment_intents
WHERE customer_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;