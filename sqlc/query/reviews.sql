-- name: DeleteReviews :exec
DELETE FROM reviews WHERE id = $1;