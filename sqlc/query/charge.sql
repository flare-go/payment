-- name: UpsertCharge :exec
INSERT INTO charges (
    id, customer_id, payment_intent_id, amount, currency, status, paid, refunded,
    failure_code, failure_message
) VALUES (
             $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
         )
ON CONFLICT (id) DO UPDATE SET
                                      id = EXCLUDED.id,
                                      payment_intent_id = EXCLUDED.payment_intent_id,
                                      amount = EXCLUDED.amount,
                                      currency = EXCLUDED.currency,
                                      status = EXCLUDED.status,
                                      paid = EXCLUDED.paid,
                                      refunded = EXCLUDED.refunded,
                                      failure_code = EXCLUDED.failure_code,
                                      failure_message = EXCLUDED.failure_message,
                                      updated_at = NOW();