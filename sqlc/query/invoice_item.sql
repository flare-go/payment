-- name: CreateInvoiceItem :exec
INSERT INTO invoice_items (
    invoice_id,
    amount,
    description
) VALUES (
             $1, $2, $3
         );
-- RETURNING id, invoice_id, amount, description, created_at, updated_at;

-- name: GetInvoiceItem :one
SELECT id, invoice_id, amount, description, created_at, updated_at
FROM invoice_items
WHERE id = $1 LIMIT 1;

-- name: UpdateInvoiceItem :exec
UPDATE invoice_items
SET amount = $2,
    description = $3,
    updated_at = NOW()
WHERE id = $1;
-- RETURNING id, invoice_id, amount, description, created_at, updated_at;

-- name: DeleteInvoiceItem :exec
DELETE FROM invoice_items WHERE id = $1;

-- name: ListInvoiceItems :many
SELECT id, invoice_id, amount, description, created_at, updated_at
FROM invoice_items
WHERE invoice_id = $1
ORDER BY created_at DESC;