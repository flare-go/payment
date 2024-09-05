-- name: CreateCustomer :one
INSERT INTO customers (
    user_id,
    balance,
    stripe_id
) VALUES (
             $1, $2, $3
         )
RETURNING id, created_at, updated_at;

-- name: GetCustomer :one
SELECT c.id, c.user_id, c.balance, c.stripe_id, c.created_at, c.updated_at,
       u.email, u.username as name
FROM customers c
         JOIN users u ON c.user_id = u.id
WHERE c.id = $1;

-- name: UpdateCustomer :exec
UPDATE customers
SET balance = $2,
    stripe_id = $3,
    updated_at = NOW()
WHERE id = $1;

-- name: DeleteCustomer :exec
DELETE FROM customers WHERE id = $1;

-- name: ListCustomers :many
SELECT c.id, c.user_id, c.balance, c.stripe_id, c.created_at, c.updated_at,
       u.email, u.username as name
FROM customers c
         JOIN users u ON c.user_id = u.id
ORDER BY c.created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateCustomerBalance :exec
UPDATE customers
SET balance = balance + $2,
    updated_at = NOW()
WHERE id = $1;