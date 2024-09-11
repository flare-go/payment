-- name: UpsertCharge :exec
INSERT INTO charges (
    id, customer_id, payment_intent_id, amount, currency, status, paid, refunded, failure_code, failure_message, created_at, updated_at
) VALUES (
             $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
         )
ON CONFLICT (id) DO UPDATE SET
                               customer_id = COALESCE($2, charges.customer_id),
                               payment_intent_id = COALESCE($3, charges.payment_intent_id),
                               amount = COALESCE($4, charges.amount),
                               currency = COALESCE($5, charges.currency),
                               status = COALESCE($6, charges.status),
                               paid = COALESCE($7, charges.paid),
                               refunded = COALESCE($8, charges.refunded),
                               failure_code = COALESCE($9, charges.failure_code),
                               failure_message = COALESCE($10, charges.failure_message),
                               created_at = COALESCE($11, charges.created_at),
                               updated_at = $12;
