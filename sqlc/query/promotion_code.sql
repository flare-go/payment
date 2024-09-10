-- name: DeletePromotionCodes :exec
DELETE FROM promotion_codes WHERE id = $1;