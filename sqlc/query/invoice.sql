-- name: CreateInvoice :exec
INSERT INTO invoices (
    id,
    customer_id,
    subscription_id,
    status,
    currency,
    amount_due,
    amount_paid,
    amount_remaining,
    due_date,
    paid_at
) VALUES (
             $1, $2, $3, $4, $5, $6, $7, $8, $9,$10
         );
-- RETURNING id, customer_id, subscription_id, status, currency, amount_due, amount_paid, amount_remaining, due_date, paid_at, stripe_id, created_at, updated_at;

-- name: GetInvoice :one
SELECT id, customer_id, subscription_id, status, currency, amount_due, amount_paid, amount_remaining, due_date, paid_at, created_at, updated_at
FROM invoices
WHERE id = $1 LIMIT 1;

-- name: UpdateInvoice :exec
UPDATE invoices
SET status = $2,
    amount_paid = $3,
    amount_remaining = $4,
    paid_at = $5,
    updated_at = NOW()
WHERE id = $1;
-- RETURNING id, customer_id, subscription_id, status, currency, amount_due, amount_paid, amount_remaining, due_date, paid_at, stripe_id, created_at, updated_at;

-- name: ListInvoicesByCustomerID :many
SELECT id, customer_id, subscription_id, status, currency, amount_due, amount_paid, amount_remaining, due_date, paid_at, created_at, updated_at
FROM invoices
WHERE customer_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListInvoices :many
SELECT id, customer_id, subscription_id, status, currency, amount_due, amount_paid, amount_remaining, due_date, paid_at, created_at, updated_at
FROM invoices
WHERE id = $1;

-- name: DeleteInvoice :exec
DELETE FROM invoices WHERE id = $1;

-- name: UpsertInvoice :exec
INSERT INTO invoices (
    id, customer_id, subscription_id, status, currency, amount_due, amount_paid,
    amount_remaining, due_date, paid_at
) VALUES (
             $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
         )
ON CONFLICT (id) DO UPDATE SET
                                      customer_id = EXCLUDED.customer_id,
                                      subscription_id = EXCLUDED.subscription_id,
                                      status = EXCLUDED.status,
                                      currency = EXCLUDED.currency,
                                      amount_due = EXCLUDED.amount_due,
                                      amount_paid = EXCLUDED.amount_paid,
                                      amount_remaining = EXCLUDED.amount_remaining,
                                      due_date = EXCLUDED.due_date,
                                      paid_at = EXCLUDED.paid_at,
                                      updated_at = NOW();
