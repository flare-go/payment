-- name: DeleteQuote :exec
DELETE FROM quotes WHERE id = $1;