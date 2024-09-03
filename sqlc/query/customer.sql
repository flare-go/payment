-- name: CreateCustomer :exec
INSERT INTO customers (
    user_id,
    balance,
    stripe_id
) VALUES (
             $1, $2, $3
         );
-- RETURNING id, user_id, balance, stripe_id, created_at, updated_at;

-- name: GetCustomer :one
SELECT id, user_id, balance, stripe_id, created_at, updated_at
FROM customers
WHERE id = $1 LIMIT 1;

-- name: UpdateCustomer :exec
UPDATE customers
SET balance = $2,
    stripe_id = $3,
    updated_at = NOW()
WHERE id = $1;

-- name: DeleteCustomer :exec
DELETE FROM customers WHERE id = $1;

-- name: ListCustomers :many
SELECT id, user_id, balance, stripe_id, created_at, updated_at
FROM customers
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateCustomerBalance :exec
UPDATE customers
SET balance = balance + $2,
    updated_at = NOW()
WHERE id = $1;