-- name: CreateCustomer :one
INSERT INTO customers (
    id,
    user_email,
    balance
) VALUES (
             $1, $2, $3
         )
RETURNING id, created_at, updated_at;

-- name: GetCustomer :one
SELECT c.id, c.balance, c.created_at, c.updated_at,
       u.email, u.username as name
FROM customers c
         JOIN users u ON c.user_email = u.email
WHERE c.id = $1;

-- name: UpdateCustomer :exec
UPDATE customers
SET balance = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: DeleteCustomer :exec
DELETE FROM customers WHERE id = $1;

-- name: ListCustomers :many
SELECT c.id, c.user_email, c.balance, c.created_at, c.updated_at,
    u.username as name
FROM customers c
         JOIN users u ON c.user_email = u.email
ORDER BY c.created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateCustomerBalance :exec
UPDATE customers
SET balance = balance + $2,
    updated_at = NOW()
WHERE id = $1;

-- name: UpsertCustomer :exec
INSERT INTO customers (id, user_email, balance, updated_at)
VALUES ($1, sqlc.narg('user_email'), sqlc.narg('balance'), $3)
ON CONFLICT (id) DO UPDATE SET
                               user_email = COALESCE(sqlc.narg('user_email'), customers.user_email),
                               balance = COALESCE(sqlc.narg('balance'), customers.balance),
                               updated_at = $3;