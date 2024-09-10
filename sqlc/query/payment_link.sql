-- name: DeletePaymentLink :exec
DELETE FROM payment_links WHERE id = $1;