-- name: DeleteTaxRate :exec
DELETE FROM tax_rates WHERE id = $1;