-- name: DeleteCheckOutSession :exec
DELETE FROM checkout_sessions WHERE id = $1;