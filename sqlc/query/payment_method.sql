-- name: CreatePaymentMethod :one
INSERT INTO payment_methods (
    customer_id,
    type,
    card_last4,
    card_brand,
    card_exp_month,
    card_exp_year,
    bank_account_last4,
    bank_account_bank_name,
    is_default,
    stripe_id
) VALUES (
             $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
         )
RETURNING id, customer_id, type, card_last4, card_brand, card_exp_month, card_exp_year, bank_account_last4, bank_account_bank_name, is_default, stripe_id, created_at, updated_at;

-- name: GetPaymentMethod :one
SELECT id, customer_id, type, card_last4, card_brand, card_exp_month, card_exp_year, bank_account_last4, bank_account_bank_name, is_default, stripe_id, created_at, updated_at
FROM payment_methods
WHERE id = $1 LIMIT 1;

-- name: UpdatePaymentMethod :one
UPDATE payment_methods
SET type = $2,
    card_last4 = $3,
    card_brand = $4,
    card_exp_month = $5,
    card_exp_year = $6,
    bank_account_last4 = $7,
    bank_account_bank_name = $8,
    is_default = $9,
    stripe_id = $10,
    updated_at = NOW()
WHERE id = $1
RETURNING id, customer_id, type, card_last4, card_brand, card_exp_month, card_exp_year, bank_account_last4, bank_account_bank_name, is_default, stripe_id, created_at, updated_at;

-- name: DeletePaymentMethod :exec
DELETE FROM payment_methods WHERE id = $1;

-- name: ListPaymentMethods :many
SELECT id, customer_id, type, card_last4, card_brand, card_exp_month, card_exp_year, bank_account_last4, bank_account_bank_name, is_default, stripe_id, created_at, updated_at
FROM payment_methods
WHERE customer_id = $1
ORDER BY is_default DESC, created_at DESC
LIMIT $2 OFFSET $3;