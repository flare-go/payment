-- name: CreateProduct :one
INSERT INTO products (
    name,
    description,
    active,
    metadata,
    stripe_id
) VALUES (
             $1, $2, $3, $4, $5
         )
RETURNING id, created_at, updated_at;

-- name: GetProduct :one
SELECT id, name, description, active, metadata, stripe_id, created_at, updated_at
FROM products
WHERE id = $1 LIMIT 1;

-- name: UpdateProduct :one
UPDATE products
SET name = $2,
    description = $3,
    active = $4,
    metadata = $5,
    stripe_id = $6,
    updated_at = NOW()
WHERE id = $1
RETURNING created_at, updated_at;

-- name: DeleteProduct :exec
DELETE FROM products WHERE id = $1;

-- name: ListProducts :many
SELECT id, name, description, active, metadata, stripe_id, created_at, updated_at
FROM products
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;