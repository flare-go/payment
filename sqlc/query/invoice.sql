-- name: CreateInvoice :one
INSERT INTO invoices (
    customer_id,
    subscription_id,
    status,
    currency,
    amount_due,
    amount_paid,
    amount_remaining,
    due_date,
    stripe_id
) VALUES (
             $1, $2, $3, $4, $5, $6, $7, $8, $9
         )
RETURNING id, customer_id, subscription_id, status, currency, amount_due, amount_paid, amount_remaining, due_date, paid_at, stripe_id, created_at, updated_at;

-- name: GetInvoice :one
SELECT id, customer_id, subscription_id, status, currency, amount_due, amount_paid, amount_remaining, due_date, paid_at, stripe_id, created_at, updated_at
FROM invoices
WHERE id = $1 LIMIT 1;

-- name: UpdateInvoice :one
UPDATE invoices
SET status = $2,
    amount_paid = $3,
    amount_remaining = $4,
    paid_at = $5,
    stripe_id = $6,
    updated_at = NOW()
WHERE id = $1
RETURNING id, customer_id, subscription_id, status, currency, amount_due, amount_paid, amount_remaining, due_date, paid_at, stripe_id, created_at, updated_at;

-- name: ListInvoices :many
SELECT id, customer_id, subscription_id, status, currency, amount_due, amount_paid, amount_remaining, due_date, paid_at, stripe_id, created_at, updated_at
FROM invoices
WHERE customer_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;